//go:build windows

package windows

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

var (
	clsidShellLink     = ole.NewGUID("{00021401-0000-0000-C000-000000000046}")
	iidShellLinkW      = ole.NewGUID("{000214F9-0000-0000-C000-000000000046}")
	iidPersistFile     = ole.NewGUID("{0000010b-0000-0000-C000-000000000046}")
	clsidShellWindows  = ole.NewGUID("{9BA05972-F6A8-11CF-A442-00A0C90A8F39}")
	iidShellWindows    = ole.NewGUID("{85CB6900-4D95-11CF-960C-0080C7F4EE85}")
	iidServiceProvider = ole.NewGUID("{6D5140C1-7436-11CE-8034-00AA006009FA}")
	sidTopLevelBrowser = ole.NewGUID("{4C96BE40-915C-11CF-99D3-00AA004AE837}")
	iidShellBrowser    = ole.NewGUID("{000214E2-0000-0000-C000-000000000046}")
	iidDispatch        = ole.NewGUID("{00020400-0000-0000-C000-000000000046}")
)

// Method numbers in the vtables used here, counting from QueryInterface.
const (
	shellLinkSetWorkingDirectory = 9
	shellLinkSetPath             = 20
	persistFileSave              = 6
	shellWindowsFindWindowSW     = 15
	serviceProviderQueryService  = 3
	shellBrowserQueryActiveView  = 15
	shellViewGetItemObject       = 15
	unknownRelease               = 2
)

const (
	swcDesktop       = 8
	swfoNeedDispatch = 1
	svgioBackground  = 0
	sFalse           = 1
)

// withCOM runs fn on a thread of its own in a single-threaded apartment,
// which the shell's objects expect. The thread ends with fn.
func withCOM(fn func() error) error {
	errc := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
			var oe *ole.OleError
			if !errors.As(err, &oe) || oe.Code() != sFalse {
				errc <- err
				return
			}
		}
		defer ole.CoUninitialize()
		errc <- fn()
	}()
	return <-errc
}

// method returns the address of method i of the COM object obj.
func method(obj unsafe.Pointer, i int) uintptr {
	vtbl := *(*unsafe.Pointer)(obj)
	return *(*uintptr)(unsafe.Add(vtbl, i*int(unsafe.Sizeof(uintptr(0)))))
}

func release(obj unsafe.Pointer) {
	syscall.SyscallN(method(obj, unknownRelease), uintptr(obj))
}

func hresult(r uintptr) error {
	if int32(r) < 0 {
		return ole.NewError(r)
	}
	return nil
}

// makeShortcut writes a .lnk at lnk that starts target in its own folder.
func makeShortcut(lnk, target string) error {
	return withCOM(func() error {
		unk, err := ole.CreateInstance(clsidShellLink, iidShellLinkW)
		if err != nil {
			return err
		}
		defer unk.Release()
		link := unsafe.Pointer(unk)

		path, err := syscall.UTF16PtrFromString(target)
		if err != nil {
			return err
		}
		dir, err := syscall.UTF16PtrFromString(filepath.Dir(target))
		if err != nil {
			return err
		}
		r, _, _ := syscall.SyscallN(method(link, shellLinkSetPath), uintptr(link), uintptr(unsafe.Pointer(path)))
		if err := hresult(r); err != nil {
			return err
		}
		r, _, _ = syscall.SyscallN(method(link, shellLinkSetWorkingDirectory), uintptr(link), uintptr(unsafe.Pointer(dir)))
		if err := hresult(r); err != nil {
			return err
		}

		pf, err := unk.QueryInterface(iidPersistFile)
		if err != nil {
			return err
		}
		defer pf.Release()
		file := unsafe.Pointer(pf)
		name, err := syscall.UTF16PtrFromString(lnk)
		if err != nil {
			return err
		}
		r, _, _ = syscall.SyscallN(method(file, persistFileSave), uintptr(file), uintptr(unsafe.Pointer(name)), 1)
		return hresult(r)
	})
}

// shellExecute starts file through the desktop's Explorer, so it runs as the
// person signed in and without the installer's elevation.
func shellExecute(file string) error {
	return withCOM(func() error {
		unk, err := ole.CreateInstance(clsidShellWindows, iidShellWindows)
		if err != nil {
			return err
		}
		defer unk.Release()
		shellWindows := unsafe.Pointer(unk)

		loc := ole.NewVariant(ole.VT_I4, 0) // CSIDL_DESKTOP
		var root ole.VARIANT
		var hwnd int32
		var desktop unsafe.Pointer
		r, _, _ := syscall.SyscallN(method(shellWindows, shellWindowsFindWindowSW), uintptr(shellWindows),
			uintptr(unsafe.Pointer(&loc)), uintptr(unsafe.Pointer(&root)), swcDesktop,
			uintptr(unsafe.Pointer(&hwnd)), swfoNeedDispatch, uintptr(unsafe.Pointer(&desktop)))
		if r == sFalse || desktop == nil {
			return errors.New("the desktop is not running")
		}
		if err := hresult(r); err != nil {
			return err
		}
		defer release(desktop)

		sp, err := (*ole.IUnknown)(desktop).QueryInterface(iidServiceProvider)
		if err != nil {
			return err
		}
		defer sp.Release()
		provider := unsafe.Pointer(sp)
		var browser unsafe.Pointer
		r, _, _ = syscall.SyscallN(method(provider, serviceProviderQueryService), uintptr(provider),
			uintptr(unsafe.Pointer(sidTopLevelBrowser)), uintptr(unsafe.Pointer(iidShellBrowser)), uintptr(unsafe.Pointer(&browser)))
		if err := hresult(r); err != nil {
			return fmt.Errorf("reaching the desktop's window: %w", err)
		}
		defer release(browser)

		var view unsafe.Pointer
		r, _, _ = syscall.SyscallN(method(browser, shellBrowserQueryActiveView), uintptr(browser), uintptr(unsafe.Pointer(&view)))
		if err := hresult(r); err != nil {
			return err
		}
		defer release(view)

		var background *ole.IDispatch
		r, _, _ = syscall.SyscallN(method(view, shellViewGetItemObject), uintptr(view), svgioBackground,
			uintptr(unsafe.Pointer(iidDispatch)), uintptr(unsafe.Pointer(&background)))
		if err := hresult(r); err != nil {
			return err
		}
		defer background.Release()

		app, err := oleutil.GetProperty(background, "Application")
		if err != nil {
			return err
		}
		shell := app.ToIDispatch()
		defer shell.Release()
		_, err = oleutil.CallMethod(shell, "ShellExecute", file, "", filepath.Dir(file), "open", 1)
		return err
	})
}
