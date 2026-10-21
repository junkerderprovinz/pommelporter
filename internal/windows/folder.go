//go:build windows

package windows

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/junkerderprovinz/pommelporter/internal/setup"
)

// lockedSDDL gives a folder to the administrators: SYSTEM and Administrators
// have full control, Users may read and run, and nothing is inherited from
// above. 0x1200a9 is what icacls calls RX.
const lockedSDDL = "O:BAD:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;0x1200a9;;;BU)"

const (
	readAndRun      = 0x1200a9
	fileDeleteChild = 0x40
	genericAll      = 0x10000000

	// takeOver are the rights that let someone rename, replace or re-permit a
	// folder, directly or through the folder above it.
	takeOver = windows.DELETE | fileDeleteChild | windows.WRITE_DAC | windows.WRITE_OWNER | genericAll

	// aceCallbackAllowed is ACCESS_ALLOWED_CALLBACK_ACE_TYPE, laid out like a
	// plain allow entry.
	aceCallbackAllowed = 9
)

const trustedInstallerSID = "S-1-5-80-956008885-3418522649-1831038044-1853292631-2271478464"

// trusted reports whether sid is one of the accounts the update task already
// trusts with the whole machine.
func trusted(sid *windows.SID) bool {
	return sid.IsWellKnown(windows.WinLocalSystemSid) ||
		sid.IsWellKnown(windows.WinBuiltinAdministratorsSid) ||
		sid.String() == trustedInstallerSID
}

func account(sid *windows.SID) string {
	name, domain, _, err := sid.LookupAccount("")
	if err != nil {
		return sid.String()
	}
	if domain == "" {
		return name
	}
	return domain + `\` + name
}

// allowed calls fn for each allow entry of sd's access list. An access list
// that is missing or null grants everything to everyone, which fn hears about
// as a nil SID.
func allowed(sd *windows.SECURITY_DESCRIPTOR, fn func(sid *windows.SID, mask uint32, flags uint8) error) error {
	dacl, _, err := sd.DACL()
	if err != nil || dacl == nil {
		return fn(nil, ^uint32(0), 0)
	}
	for i := range uint32(dacl.AceCount) {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, i, &ace); err != nil {
			return err
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE && ace.Header.AceType != aceCallbackAllowed {
			continue
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if err := fn(sid, uint32(ace.Mask), ace.Header.AceFlags); err != nil {
			return err
		}
	}
	return nil
}

// parentRisk says why a folder above the install folder would let an
// ordinary user swap the install folder, or returns nil. Entries that only
// pass on to new children are left out: the folders the installer creates
// below carry their own access list.
func parentRisk(sd *windows.SECURITY_DESCRIPTOR) error {
	owner, _, err := sd.Owner()
	if err != nil {
		return err
	}
	if !trusted(owner) {
		return fmt.Errorf("it belongs to %s", account(owner))
	}
	return allowed(sd, func(sid *windows.SID, mask uint32, flags uint8) error {
		switch {
		case sid == nil:
			return errors.New("its access list lets everyone change it")
		case flags&windows.INHERIT_ONLY_ACE != 0, trusted(sid):
			return nil
		case mask&takeOver != 0:
			return fmt.Errorf("%s may rename, replace or re-permit it", account(sid))
		}
		return nil
	})
}

// lockedRisk says how a folder falls short of lockedSDDL, or returns nil.
// Unlike a parent, the install folder itself may not let anyone else write.
func lockedRisk(sd *windows.SECURITY_DESCRIPTOR) error {
	owner, _, err := sd.Owner()
	if err != nil {
		return err
	}
	if !owner.IsWellKnown(windows.WinBuiltinAdministratorsSid) {
		return fmt.Errorf("it belongs to %s", account(owner))
	}
	control, _, err := sd.Control()
	if err != nil {
		return err
	}
	if control&windows.SE_DACL_PROTECTED == 0 {
		return errors.New("it still inherits rights from the folder above")
	}
	return allowed(sd, func(sid *windows.SID, mask uint32, _ uint8) error {
		switch {
		case sid == nil:
			return errors.New("its access list lets everyone change it")
		case trusted(sid):
			return nil
		case mask&^readAndRun != 0:
			return fmt.Errorf("%s may change what is in it", account(sid))
		}
		return nil
	})
}

// normalize turns dir into the form the apps see as their own folder: full,
// clean, without the dots and spaces Windows drops from names, and with long
// names for the part that exists.
func normalize(dir string) (string, error) {
	p, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return "", err
	}
	buf := make([]uint16, windows.MAX_LONG_PATH)
	n, err := windows.GetFullPathName(p, uint32(len(buf)), &buf[0], nil)
	if err != nil {
		return "", err
	}
	full := filepath.Clean(windows.UTF16ToString(buf[:n]))

	existing, rest := full, ""
	for !exists(existing) {
		up := filepath.Dir(existing)
		if up == existing {
			return full, nil
		}
		rest = filepath.Join(filepath.Base(existing), rest)
		existing = up
	}
	p, _ = windows.UTF16PtrFromString(existing)
	n, err = windows.GetLongPathName(p, &buf[0], uint32(len(buf)))
	if err != nil {
		return "", err
	}
	return filepath.Join(windows.UTF16ToString(buf[:n]), rest), nil
}

func exists(path string) bool {
	_, err := attributes(path)
	return err == nil
}

// attributes reads path's attributes without following a link there.
func attributes(path string) (uint32, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	return windows.GetFileAttributes(p)
}

func notFound(err error) bool {
	return errors.Is(err, windows.ERROR_FILE_NOT_FOUND) || errors.Is(err, windows.ERROR_PATH_NOT_FOUND)
}

func isLink(attrs uint32) bool { return attrs&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 }

func isDir(attrs uint32) bool { return attrs&windows.FILE_ATTRIBUTE_DIRECTORY != 0 }

func hasACLs(fsName string) bool {
	return strings.EqualFold(fsName, "NTFS") || strings.EqualFold(fsName, "ReFS")
}

func refuse(code, format string, args ...any) setup.DirCheck {
	return setup.DirCheck{Code: code, Detail: fmt.Sprintf(format, args...)}
}

// checkDir applies the folder rules to a normalized dir. earlier tells
// whether a folder that is not empty holds an earlier copy of the app.
func checkDir(dir string, earlier func(string) bool) setup.DirCheck {
	vol := filepath.VolumeName(dir)
	if len(vol) != 2 || vol[1] != ':' {
		return refuse(setup.DirNotLocal, "%s is not on a drive of this computer", dir)
	}
	root := vol + `\`
	rootp, _ := windows.UTF16PtrFromString(root)
	if windows.GetDriveType(rootp) != windows.DRIVE_FIXED {
		return refuse(setup.DirNotLocal, "%s is not a fixed drive of this computer", root)
	}
	// A drive letter made with subst names a folder, not a volume, and the
	// update task would not see it.
	var volName [windows.MAX_PATH]uint16
	if err := windows.GetVolumeNameForVolumeMountPoint(rootp, &volName[0], uint32(len(volName))); err != nil {
		return refuse(setup.DirNotLocal, "%s is not a volume of this computer", root)
	}
	var fsName [windows.MAX_PATH]uint16
	if err := windows.GetVolumeInformation(rootp, nil, 0, nil, nil, nil, &fsName[0], uint32(len(fsName))); err != nil {
		return refuse(setup.DirNoACL, "the file system of %s is unknown: %v", root, err)
	}
	if fs := windows.UTF16ToString(fsName[:]); !hasACLs(fs) {
		return refuse(setup.DirNoACL, "%s uses %s, which keeps no access rights", root, fs)
	}
	if dir == root {
		return refuse(setup.DirNotEmpty, "%s is the top folder of a drive", dir)
	}

	var chain []string
	for p := dir; ; p = filepath.Dir(p) {
		chain = append(chain, p)
		if p == root {
			break
		}
	}
	slices.Reverse(chain)
	var found []string
	for _, p := range chain {
		attrs, err := attributes(p)
		if notFound(err) {
			break
		}
		if err != nil {
			return refuse(setup.DirUnsafeParent, "%s cannot be read: %v", p, err)
		}
		if isLink(attrs) {
			return refuse(setup.DirLink, "%s is a link to another place", p)
		}
		if !isDir(attrs) {
			return refuse(setup.DirFile, "%s is a file", p)
		}
		found = append(found, p)
	}

	for _, p := range found {
		if p == dir {
			break
		}
		sd, err := windows.GetNamedSecurityInfo(p, windows.SE_FILE_OBJECT,
			windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
		if err != nil {
			return refuse(setup.DirUnsafeParent, "the rights on %s cannot be read: %v", p, err)
		}
		if err := parentRisk(sd); err != nil {
			return refuse(setup.DirUnsafeParent, "%s is above the folder and %v", p, err)
		}
	}

	if len(found) == len(chain) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return refuse(setup.DirNotEmpty, "%s cannot be read: %v", dir, err)
		}
		if len(entries) > 0 && !earlier(dir) {
			return refuse(setup.DirNotEmpty, "%s already holds other files", dir)
		}
	}
	return setup.DirCheck{OK: true}
}

// enablePrivileges lets the installer take over a folder whatever its access
// list says, as icacls /setowner does.
func enablePrivileges() error {
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_ADJUST_PRIVILEGES|windows.TOKEN_QUERY, &token); err != nil {
		return err
	}
	defer token.Close()
	for _, name := range []string{"SeRestorePrivilege", "SeTakeOwnershipPrivilege"} {
		var luid windows.LUID
		if err := windows.LookupPrivilegeValue(nil, windows.StringToUTF16Ptr(name), &luid); err != nil {
			return err
		}
		tp := windows.Tokenprivileges{PrivilegeCount: 1}
		tp.Privileges[0] = windows.LUIDAndAttributes{Luid: luid, Attributes: windows.SE_PRIVILEGE_ENABLED}
		if err := windows.AdjustTokenPrivileges(token, false, &tp, 0, nil, nil); err != nil {
			return err
		}
	}
	return nil
}

func lockedDescriptor() (*windows.SECURITY_DESCRIPTOR, error) {
	return windows.SecurityDescriptorFromString(lockedSDDL)
}

// createLocked creates path with lockedSDDL in one step, so nobody else ever
// holds rights on it. It fails when path exists.
func createLocked(path string) error {
	sd, err := lockedDescriptor()
	if err != nil {
		return err
	}
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	sa := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: sd}
	return windows.CreateDirectory(p, &sa)
}

// lockDir makes path a folder only administrators can change: created that
// way when it is missing, taken over when it exists. The work goes through a
// handle to the folder itself, never through a link, and ends with a check
// that the rights hold and the path still names the same folder.
func lockDir(path string) error {
	if err := enablePrivileges(); err != nil {
		return err
	}
	if err := createLocked(path); err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return err
	}
	h, info, err := openNoFollow(path, windows.READ_CONTROL|windows.WRITE_DAC|windows.WRITE_OWNER)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	if isLink(info.FileAttributes) {
		return fmt.Errorf("%s is a link to another place", path)
	}
	if !isDir(info.FileAttributes) {
		return fmt.Errorf("%s is a file", path)
	}

	sd, err := lockedDescriptor()
	if err != nil {
		return err
	}
	owner, _, err := sd.Owner()
	if err != nil {
		return err
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return err
	}
	err = windows.SetSecurityInfo(h, windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		owner, nil, dacl, nil)
	if err != nil {
		return fmt.Errorf("setting the rights on %s: %w", path, err)
	}
	got, err := windows.GetSecurityInfo(h, windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	if err := lockedRisk(got); err != nil {
		return fmt.Errorf("the rights on %s did not hold: %v", path, err)
	}

	h2, again, err := openNoFollow(path, windows.READ_CONTROL)
	if err != nil {
		return err
	}
	windows.CloseHandle(h2)
	if again.VolumeSerialNumber != info.VolumeSerialNumber ||
		again.FileIndexHigh != info.FileIndexHigh || again.FileIndexLow != info.FileIndexLow {
		return fmt.Errorf("%s was replaced while its rights were set", path)
	}
	return nil
}

// openNoFollow opens path itself, a link as the link and not where it leads.
func openNoFollow(path string, access uint32) (windows.Handle, windows.ByHandleFileInformation, error) {
	var info windows.ByHandleFileInformation
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, info, err
	}
	h, err := windows.CreateFile(p, access,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return 0, info, err
	}
	if err := windows.GetFileInformationByHandle(h, &info); err != nil {
		windows.CloseHandle(h)
		return 0, info, err
	}
	return h, info, nil
}

// lockChain locks dir and every missing folder above it. A new folder would
// otherwise inherit what the drive's top folder hands down, which on a second
// drive lets every user rename it.
func lockChain(dir string) error {
	var missing []string
	for p := filepath.Dir(dir); !exists(p); p = filepath.Dir(p) {
		missing = append(missing, p)
	}
	for i := len(missing) - 1; i >= 0; i-- {
		if err := lockDir(missing[i]); err != nil {
			return err
		}
	}
	return lockDir(dir)
}
