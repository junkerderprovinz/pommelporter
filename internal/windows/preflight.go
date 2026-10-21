//go:build windows

package windows

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/junkerderprovinz/pommelporter/internal/setup"
)

const (
	webView2Client    = `Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`
	webView2Bootstrap = "https://go.microsoft.com/fwlink/p/?LinkId=2124703"
)

var (
	procEnumThreadWindows = user32.NewProc("EnumThreadWindows")
	procGetClassNameW     = user32.NewProc("GetClassNameW")
)

// Prepare does what has to happen before the installer's window can open: it
// refuses a payload built for another processor, and brings the WebView2
// runtime that both the window and the app need. A silent uninstall needs
// neither.
func Prepare(app setup.App, silent, uninstall bool) error {
	if silent && uninstall {
		return nil
	}
	if !uninstall {
		if err := checkArch(app); err != nil {
			return fail(app, silent, err)
		}
	}
	if hasWebView2() {
		return nil
	}
	install := func() error { return installWebView2(context.Background()) }
	if silent {
		return install()
	}
	return fail(app, silent, whileShowing(app.Name,
		"Installing the Microsoft Edge WebView2 Runtime, which "+app.Name+" needs. This takes a minute or two.", install))
}

func fail(app setup.App, silent bool, err error) error {
	if err != nil && !silent {
		message(app.Name, err.Error(), windows.MB_ICONERROR)
	}
	return err
}

// The machine types IsWow64Process2 and a program's PE header name.
const (
	machineAMD64 = 0x8664
	machineARM64 = 0xaa64
)

// checkArch compares the processor the payload's program was built for with
// the one Windows runs on, so an x64 build is not installed on ARM, where it
// would run emulated and update itself to the wrong build.
func checkArch(app setup.App) error {
	f, err := app.Payload.Open(app.Executable)
	if err != nil {
		return err
	}
	defer f.Close()
	built, err := peMachine(f)
	if err != nil {
		return fmt.Errorf("%s: %w", app.Executable, err)
	}
	var process, native uint16
	if err := windows.IsWow64Process2(windows.CurrentProcess(), &process, &native); err != nil {
		return err
	}
	if built != native {
		return fmt.Errorf("this %s is built for %s computers, and this one is %s", app.Name, archName(built), archName(native))
	}
	return nil
}

func archName(machine uint16) string {
	switch machine {
	case machineAMD64:
		return "x64"
	case machineARM64:
		return "ARM64"
	}
	return fmt.Sprintf("type %#x", machine)
}

// peMachine reads the machine field of a Windows program's header.
func peMachine(r io.Reader) (uint16, error) {
	head := make([]byte, 4096)
	n, err := io.ReadFull(r, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return 0, err
	}
	head = head[:n]
	if len(head) < 0x40 || string(head[:2]) != "MZ" {
		return 0, errors.New("not a Windows program")
	}
	off := int(binary.LittleEndian.Uint32(head[0x3c:]))
	if off < 0 || off+6 > len(head) || string(head[off:off+4]) != "PE\x00\x00" {
		return 0, errors.New("not a Windows program")
	}
	return binary.LittleEndian.Uint16(head[off+4:]), nil
}

// hasWebView2 looks where Wails' NSIS macro and Microsoft's guide look: the
// runtime registers a version both for all users and for one.
func hasWebView2() bool {
	for _, k := range []struct {
		root registry.Key
		path string
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\` + webView2Client},
		{registry.CURRENT_USER, `Software\` + webView2Client},
	} {
		if v := readString(k.root, k.path, "pv"); v != "" && v != "0.0.0.0" {
			return true
		}
	}
	return false
}

// installWebView2 downloads Microsoft's bootstrapper into a folder only
// administrators can change, checks its signature and runs it.
func installWebView2(ctx context.Context) error {
	dir, err := privateTemp()
	if err != nil {
		return err
	}
	defer removeTree(dir)
	setupExe := filepath.Join(dir, "MicrosoftEdgeWebview2Setup.exe")
	if err := download(ctx, webView2Bootstrap, setupExe); err != nil {
		return fmt.Errorf("downloading the WebView2 Runtime: %w", err)
	}
	if err := verifySignature(setupExe); err != nil {
		return fmt.Errorf("the WebView2 setup is not signed as it should be: %w", err)
	}
	if err := exec.CommandContext(ctx, setupExe, "/silent", "/install").Run(); err != nil {
		return fmt.Errorf("installing the WebView2 Runtime: %w", err)
	}
	if !hasWebView2() {
		return errors.New("the WebView2 Runtime is still missing after its setup ran")
	}
	return nil
}

func download(ctx context.Context, url, path string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s answered %s", url, resp.Status)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// verifySignature asks Windows whether path carries a valid Authenticode
// signature from a trusted publisher.
func verifySignature(path string) error {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	file := &windows.WinTrustFileInfo{Size: uint32(unsafe.Sizeof(windows.WinTrustFileInfo{})), FilePath: p}
	data := &windows.WinTrustData{
		Size:                            uint32(unsafe.Sizeof(windows.WinTrustData{})),
		UIChoice:                        windows.WTD_UI_NONE,
		RevocationChecks:                windows.WTD_REVOKE_NONE,
		UnionChoice:                     windows.WTD_CHOICE_FILE,
		StateAction:                     windows.WTD_STATEACTION_VERIFY,
		FileOrCatalogOrBlobOrSgnrOrCert: unsafe.Pointer(file),
	}
	verr := windows.WinVerifyTrustEx(windows.InvalidHWND, &windows.WINTRUST_ACTION_GENERIC_VERIFY_V2, data)
	data.StateAction = windows.WTD_STATEACTION_CLOSE
	windows.WinVerifyTrustEx(windows.InvalidHWND, &windows.WINTRUST_ACTION_GENERIC_VERIFY_V2, data)
	return verr
}

func message(title, text string, icon uint32) {
	t, _ := windows.UTF16PtrFromString(title)
	m, _ := windows.UTF16PtrFromString(text)
	windows.MessageBox(0, m, t, windows.MB_OK|icon|windows.MB_SETFOREGROUND)
}

// whileShowing shows a native note while work runs and closes it when work
// is done. Closing the note early only hides it.
func whileShowing(title, text string, work func() error) error {
	tid := make(chan uint32)
	shown := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		tid <- windows.GetCurrentThreadId()
		message(title, text, windows.MB_ICONINFORMATION)
		close(shown)
	}()
	thread := <-tid
	err := work()

	closeBox := syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		var class [16]uint16
		procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&class[0])), uintptr(len(class)))
		if windows.UTF16ToString(class[:]) == "#32770" {
			procPostMessageW.Call(hwnd, wmClose, 0, 0)
		}
		return 1
	})
	for {
		procEnumThreadWindows.Call(uintptr(thread), closeBox, 0)
		select {
		case <-shown:
			return err
		case <-time.After(50 * time.Millisecond):
		}
	}
}
