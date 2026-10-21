//go:build windows

package windows

import (
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	uninstallRoot = `Software\Microsoft\Windows\CurrentVersion\Uninstall\`
	runKey        = `Software\Microsoft\Windows\CurrentVersion\Run`
)

var procRegDeleteTree = windows.NewLazySystemDLL("advapi32.dll").NewProc("RegDeleteTreeW")

func readString(root registry.Key, path, name string) string {
	k, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	s, _, err := k.GetStringValue(name)
	if err != nil {
		return ""
	}
	return s
}

func readDWORD(root registry.Key, path, name string) (uint64, bool) {
	k, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		return 0, false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue(name)
	return v, err == nil
}

func keyExists(root registry.Key, path string) bool {
	k, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	k.Close()
	return true
}

// deleteKey removes path with everything below it, as NSIS's DeleteRegKey
// does. A key that is not there is no error.
func deleteKey(root registry.Key, path string) error {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	r, _, _ := procRegDeleteTree.Call(uintptr(root), uintptr(unsafe.Pointer(p)))
	if r != 0 && windows.Errno(r) != windows.ERROR_FILE_NOT_FOUND {
		return windows.Errno(r)
	}
	return nil
}

// installDirOf reads the folder an uninstall entry belongs to. Wails' own
// installers recorded only the program as DisplayIcon.
func installDirOf(root registry.Key, key string) string {
	if dir := readString(root, key, "InstallLocation"); dir != "" {
		return filepath.Clean(dir)
	}
	if exe := iconPath(readString(root, key, "DisplayIcon")); exe != "" {
		return filepath.Dir(exe)
	}
	return ""
}

// iconPath strips the quotes and icon index a DisplayIcon may carry.
func iconPath(icon string) string {
	if i := strings.LastIndex(icon, ","); i > 0 && !strings.ContainsAny(icon[i:], `\/`) {
		icon = icon[:i]
	}
	icon = strings.Trim(icon, `" `)
	if icon == "" {
		return ""
	}
	return filepath.Clean(icon)
}

// firstToken returns the program a command line starts.
func firstToken(cmdline string) string {
	cmdline = strings.TrimSpace(cmdline)
	if rest, ok := strings.CutPrefix(cmdline, `"`); ok {
		exe, _, _ := strings.Cut(rest, `"`)
		return exe
	}
	exe, _, _ := strings.Cut(cmdline, " ")
	return exe
}
