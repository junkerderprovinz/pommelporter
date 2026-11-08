//go:build windows

package windows

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

// The update task runs as SYSTEM and writes its log into this folder, so a
// user who could change the folder could lead that write elsewhere.
func TestTheUpdateFolderLetsUsersChangeOnlyTheSettings(t *testing.T) {
	needAdmin(t)
	dir := filepath.Join(safeBase(t), "App")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "update.log.tmp"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	user, err := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;" + someUser + ")")
	if err != nil {
		t.Fatal(err)
	}
	dacl, _, _ := user.DACL()
	if err := windows.SetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil); err != nil {
		t.Fatal(err)
	}

	if err := prepareData(dir, nil, "en"); err != nil {
		t.Fatal(err)
	}
	lockedOK(t, dir)
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != settingsFile {
		t.Fatalf("the folder holds %v", entries)
	}

	sd, err := windows.GetNamedSecurityInfo(filepath.Join(dir, settingsFile), windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	err = allowed(sd, func(sid *windows.SID, mask uint32, _ uint8) error {
		if sid != nil && !trusted(sid) && mask&(windows.DELETE|windows.WRITE_DAC|windows.WRITE_OWNER) != 0 {
			t.Errorf("%s may delete or re-permit the settings", account(sid))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
