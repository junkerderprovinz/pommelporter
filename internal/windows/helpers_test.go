//go:build windows

package windows

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func elevated() bool { return windows.GetCurrentProcessToken().IsElevated() }

func needAdmin(t *testing.T) {
	t.Helper()
	if !elevated() {
		t.Skip("needs an elevated test run")
	}
}

func randomName(prefix string) string {
	var b [6]byte
	rand.Read(b[:])
	return prefix + hex.EncodeToString(b[:])
}

// testKey returns a fresh key under HKCU, standing in for HKLM or HKCU, and
// its path for reg.exe.
func testKey(t *testing.T) (registry.Key, string) {
	t.Helper()
	path := `Software\PommelPorterTest\` + randomName("")
	k, _, err := registry.CreateKey(registry.CURRENT_USER, path, registry.ALL_ACCESS)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		k.Close()
		deleteKey(registry.CURRENT_USER, path)
		registry.DeleteKey(registry.CURRENT_USER, `Software\PommelPorterTest`)
	})
	return k, `HKCU\` + path
}

func setString(t *testing.T, root registry.Key, path, name, value string) {
	t.Helper()
	k, _, err := registry.CreateKey(root, path, registry.SET_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	defer k.Close()
	if err := k.SetStringValue(name, value); err != nil {
		t.Fatal(err)
	}
}

type fakeTasks struct{ calls [][]string }

func (f *fakeTasks) run(args ...string) error {
	f.calls = append(f.calls, args)
	return nil
}

func testPlatform(t *testing.T) (*platform, *fakeTasks) {
	t.Helper()
	base := t.TempDir()
	dir := func(name string) string {
		d := filepath.Join(base, name)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		return d
	}
	machine, _ := testKey(t)
	user, _ := testKey(t)
	tasks := &fakeTasks{}
	return &platform{
		machine: machine,
		user:    user,
		places: places{
			programFiles:   dir("ProgramFiles"),
			programData:    dir("ProgramData"),
			commonPrograms: dir("CommonPrograms"),
			publicDesktop:  dir("PublicDesktop"),
			userPrograms:   dir("UserPrograms"),
			userDesktop:    dir("UserDesktop"),
			appData:        dir("AppData"),
		},
		schtasks: tasks.run,
	}, tasks
}

func junction(t *testing.T, link, target string) {
	t.Helper()
	out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput()
	if err != nil {
		t.Fatalf("mklink /J: %v: %s", err, out)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fakeProgram is the start of a Windows program built for machine, enough
// for its header to be read.
func fakeProgram(machine uint16) []byte {
	b := make([]byte, 0x200)
	copy(b, "MZ")
	binary.LittleEndian.PutUint32(b[0x3c:], 0x80)
	copy(b[0x80:], "PE\x00\x00")
	binary.LittleEndian.PutUint16(b[0x84:], machine)
	return b
}

func nativeMachine(t *testing.T) uint16 {
	t.Helper()
	var process, native uint16
	if err := windows.IsWow64Process2(windows.CurrentProcess(), &process, &native); err != nil {
		t.Fatal(err)
	}
	return native
}

// safeBase creates an empty folder under ProgramData, whose folders above
// are ones ordinary users cannot swap on a usual Windows installation.
func safeBase(t *testing.T) string {
	t.Helper()
	pd := known(windows.FOLDERID_ProgramData)
	for _, p := range []string{filepath.VolumeName(pd) + `\`, pd} {
		sd, err := windows.GetNamedSecurityInfo(p, windows.SE_FILE_OBJECT,
			windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
		if err != nil {
			t.Fatal(err)
		}
		if err := parentRisk(sd); err != nil {
			t.Skipf("%s is not a safe parent on this machine: %v", p, err)
		}
	}
	base := filepath.Join(pd, randomName("pommelporter-test-"))
	if err := os.Mkdir(base, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { removeTree(base) })
	return base
}
