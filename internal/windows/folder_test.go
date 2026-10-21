//go:build windows

package windows

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/junkerderprovinz/pommelporter/internal/setup"
)

const someUser = "S-1-5-21-1-2-3-1001"

func descriptor(t *testing.T, sddl string) *windows.SECURITY_DESCRIPTOR {
	t.Helper()
	sd, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		t.Fatalf("%s: %v", sddl, err)
	}
	return sd
}

func TestParentRisk(t *testing.T) {
	for _, tc := range []struct {
		name string
		sddl string
		safe bool
	}{
		{"administrators and users read", "O:BAD:(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;0x1200a9;;;BU)", true},
		{"system drive top folder",
			"O:" + trustedInstallerSID + "D:PAI(A;;0x4;;;AU)(A;OICIIO;SDGXGWGR;;;AU)(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;0x1200a9;;;BU)", true},
		{"second drive grants create folders only", "O:BAD:(A;;0x4;;;AU)(A;OICI;FA;;;BA)", true},
		{"owned by system", "O:SYD:(A;;FA;;;SY)", true},
		{"owned by trusted installer", "O:" + trustedInstallerSID + "D:(A;;FA;;;" + trustedInstallerSID + ")", true},
		{"creator owner on inherit-only entries", "O:BAD:(A;OICIIO;GA;;;CO)(A;OICI;FA;;;BA)", true},
		{"users write without delete", "O:BAD:(A;;FRFW;;;BU)", true},
		{"deny entries change nothing", "O:BAD:(D;;FA;;;AU)(A;;FA;;;BA)", true},
		{"owned by a user", "O:" + someUser + "D:(A;;FA;;;BA)", false},
		{"users may delete", "O:BAD:(A;;SD;;;AU)", false},
		{"users may delete children", "O:BAD:(A;;0x40;;;BU)", false},
		{"users may change the rights", "O:BAD:(A;;WD;;;BU)", false},
		{"users may take ownership", "O:BAD:(A;;WO;;;BU)", false},
		{"users hold generic all", "O:BAD:(A;;GA;;;BU)", false},
		{"a user has full control", "O:BAD:(A;OICI;FA;;;" + someUser + ")", false},
		{"creator owner on an effective entry", "O:BAD:(A;;FA;;;CO)", false},
		{"null access list", "O:BAD:NO_ACCESS_CONTROL", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := parentRisk(descriptor(t, tc.sddl))
			if tc.safe && err != nil {
				t.Errorf("refused: %v", err)
			}
			if !tc.safe && err == nil {
				t.Error("accepted")
			}
		})
	}
}

func TestLockedRisk(t *testing.T) {
	for _, tc := range []struct {
		name string
		sddl string
		safe bool
	}{
		{"locked", lockedSDDL, true},
		{"still inherits", "O:BAD:(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;0x1200a9;;;BU)", false},
		{"owned by system", "O:SYD:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)", false},
		{"users may write", "O:BAD:P(A;OICI;FA;;;BA)(A;OICI;FRFW;;;BU)", false},
		{"users may write in new files", "O:BAD:P(A;OICI;FA;;;BA)(A;OICIIO;FW;;;BU)", false},
		{"users may add folders", "O:BAD:P(A;OICI;FA;;;BA)(A;;0x4;;;AU)", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := lockedRisk(descriptor(t, tc.sddl), nil)
			if tc.safe && err != nil {
				t.Errorf("refused: %v", err)
			}
			if !tc.safe && err == nil {
				t.Error("accepted")
			}
		})
	}
}

func TestHasACLs(t *testing.T) {
	for fs, want := range map[string]bool{"NTFS": true, "ReFS": true, "FAT32": false, "exFAT": false, "FAT": false} {
		if got := hasACLs(fs); got != want {
			t.Errorf("hasACLs(%s) = %v", fs, got)
		}
	}
}

func checkCode(t *testing.T, p *platform, dir, want string) {
	t.Helper()
	chk := p.CheckDir(appFixture(), dir)
	if want == "" {
		if !chk.OK {
			t.Errorf("CheckDir(%s) refused it: %s %s", dir, chk.Code, chk.Detail)
		}
		return
	}
	if chk.OK || chk.Code != want {
		t.Errorf("CheckDir(%s) = %+v, want code %s", dir, chk, want)
	}
}

func TestCheckDirRefusesRelativePaths(t *testing.T) {
	p, _ := testPlatform(t)
	checkCode(t, p, `Apps\Prog`, setup.DirRelative)
	checkCode(t, p, `\Apps\Prog`, setup.DirRelative)
}

func TestCheckDirRefusesNetworkPaths(t *testing.T) {
	p, _ := testPlatform(t)
	checkCode(t, p, `\\localhost\C$\Prog`, setup.DirNotLocal)
}

func TestCheckDirRefusesDriveTop(t *testing.T) {
	p, _ := testPlatform(t)
	checkCode(t, p, os.Getenv("SystemDrive")+`\`, setup.DirNotEmpty)
}

func TestCheckDirRefusesFiles(t *testing.T) {
	p, _ := testPlatform(t)
	file := filepath.Join(t.TempDir(), "file")
	writeFile(t, file, "x")
	checkCode(t, p, file, setup.DirFile)
	checkCode(t, p, filepath.Join(file, "Prog"), setup.DirFile)
}

func TestCheckDirRefusesJunctions(t *testing.T) {
	p, _ := testPlatform(t)
	tmp := t.TempDir()
	real := filepath.Join(tmp, "real")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(tmp, "link")
	junction(t, link, real)
	checkCode(t, p, link, setup.DirLink)
	checkCode(t, p, filepath.Join(link, "Prog"), setup.DirLink)
}

func TestFoldersUsersCanSwapAreAcceptedAsExposed(t *testing.T) {
	p, _ := testPlatform(t)
	// The folders a user's temp folder sits in belong to that user.
	under := filepath.Join(t.TempDir(), "Prog")
	// Whoever may delete inside a folder can move the install folder out of it.
	below := filepath.Join(safeBase(t), "below")
	if err := os.Mkdir(below, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("icacls", below, "/grant", "*S-1-5-32-545:(DC)").CombinedOutput(); err != nil {
		t.Fatalf("icacls: %v: %s", err, out)
	}
	for _, dir := range []string{under, filepath.Join(below, "Prog")} {
		checkCode(t, p, dir, "")
		if !exposed(dir) {
			t.Errorf("%s is not seen as exposed", dir)
		}
	}
}

func TestAFolderUnderSafeParentsIsNotExposed(t *testing.T) {
	// Run without elevation, the test's own folder belongs to the user.
	needAdmin(t)
	if dir := filepath.Join(safeBase(t), "Prog"); exposed(dir) {
		t.Errorf("%s is seen as exposed", dir)
	}
}

func TestCheckDirAcceptsAFolderUnderSafeParents(t *testing.T) {
	p, _ := testPlatform(t)
	base := safeBase(t)
	checkCode(t, p, base, "")
	os.Remove(base)
	checkCode(t, p, base, "")
}

func TestCheckDirRefusesAFolderWithOtherFiles(t *testing.T) {
	p, _ := testPlatform(t)
	base := safeBase(t)
	writeFile(t, filepath.Join(base, "notes.txt"), "x")
	checkCode(t, p, base, setup.DirNotEmpty)
}

func TestCheckDirAcceptsTheEarlierInstall(t *testing.T) {
	p, _ := testPlatform(t)
	app := appFixture()
	base := safeBase(t)
	writeFile(t, filepath.Join(base, app.Executable), "x")
	setString(t, p.machine, entryKey(app), "InstallLocation", strings.ToUpper(base))
	checkCode(t, p, base, "")
}

func TestCheckDirAcceptsAFolderHoldingAnOlderInstall(t *testing.T) {
	p, _ := testPlatform(t)
	app := appFixture()
	base := safeBase(t)
	old := filepath.Join(base, app.Name, app.Executable)
	writeFile(t, old, "x")
	setString(t, p.machine, app.Legacy.MachineKeys[0], "DisplayIcon", old)
	checkCode(t, p, base, "")
}

func TestNormalizeDropsWhatWindowsDrops(t *testing.T) {
	tmp := t.TempDir()
	got, err := normalize(filepath.Join(tmp, "Prog. ") + `\..\Prog.`)
	if err != nil {
		t.Fatal(err)
	}
	long, _ := normalize(tmp)
	if want := filepath.Join(long, "Prog"); got != want {
		t.Errorf("normalize = %s, want %s", got, want)
	}
}

func TestNormalizeUsesLongNames(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "A long folder name")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p, _ := windows.UTF16PtrFromString(dir)
	buf := make([]uint16, windows.MAX_PATH)
	n, err := windows.GetShortPathName(p, &buf[0], uint32(len(buf)))
	short := windows.UTF16ToString(buf[:n])
	if err != nil || strings.EqualFold(short, dir) {
		t.Skip("the volume keeps no short names")
	}
	got, err := normalize(filepath.Join(short, "Prog"))
	if err != nil {
		t.Fatal(err)
	}
	// The temp folder itself may be spelled short, as on CI runners.
	n, err = windows.GetLongPathName(p, &buf[0], uint32(len(buf)))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(windows.UTF16ToString(buf[:n]), "Prog"); !strings.EqualFold(got, want) {
		t.Errorf("normalize = %s, want %s", got, want)
	}
}

func TestLockDirRefusesAJunction(t *testing.T) {
	tmp := t.TempDir()
	real := filepath.Join(tmp, "real")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(tmp, "link")
	junction(t, link, real)
	if err := lockDir(link, nil); err == nil {
		t.Fatal("lockDir took a junction")
	}
}

func TestLockDirRefusesAFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	writeFile(t, file, "x")
	if err := lockDir(file, nil); err == nil {
		t.Fatal("lockDir took a file")
	}
}

func lockedOK(t *testing.T, dir string) {
	t.Helper()
	sd, err := windows.GetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	if err := lockedRisk(sd, nil); err != nil {
		t.Errorf("%s: %v", dir, err)
	}
}

func TestLockDirTakesOverAUsersFolder(t *testing.T) {
	needAdmin(t)
	dir := filepath.Join(safeBase(t), "Prog")
	if err := os.Mkdir(dir, 0o755); err != nil {
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
	if err := lockDir(dir, nil); err != nil {
		t.Fatal(err)
	}
	lockedOK(t, dir)
}

func TestLockChainLocksMissingFolders(t *testing.T) {
	needAdmin(t)
	base := safeBase(t)
	dir := filepath.Join(base, "a", "b")
	if err := lockChain(dir, nil); err != nil {
		t.Fatal(err)
	}
	lockedOK(t, filepath.Join(base, "a"))
	lockedOK(t, dir)
}
