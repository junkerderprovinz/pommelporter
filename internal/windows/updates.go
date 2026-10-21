//go:build windows

package windows

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"unicode/utf16"

	"golang.org/x/sys/windows"

	"github.com/junkerderprovinz/pommelporter/internal/setup"
)

const settingsFile = "settings.json"

// prepareData readies the folder under ProgramData that holds the update
// switch and the task's log. It belongs to the administrators and only
// settings.json is open to every user, so the app's settings can change the
// switch; anything else a user could write there, the task running as SYSTEM
// could be tricked into writing somewhere else. What an earlier folder held
// is cleared, keeping only the switch. An error means the folder is not safe
// for the task.
func prepareData(dir string) error {
	auto := readAutoUpdate(filepath.Join(dir, settingsFile))

	// A junction in place of the folder would lead everything below elsewhere.
	if attrs, err := attributes(dir); err == nil && isLink(attrs) {
		if err := os.Remove(dir); err != nil {
			return err
		}
	}
	// A user may have made the folder before the first install and given
	// themselves rights on it. It is cleared only once no user can change it.
	if err := lockDir(dir); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		// A sub-folder goes only when it is empty or a link, which Remove
		// unlinks instead of following. A full one a user left there is
		// harmless unless it has the log's name.
		os.Remove(filepath.Join(dir, e.Name()))
	}
	if attrs, err := attributes(filepath.Join(dir, "update.log")); err == nil && isDir(attrs) {
		return errors.New("update.log in " + dir + " is a folder")
	}

	settings := filepath.Join(dir, settingsFile)
	if err := placeFile(settings, strings.NewReader(fmt.Sprintf("{\"autoUpdate\":%t}\r\n", auto))); err != nil {
		return err
	}
	sd, err := windows.SecurityDescriptorFromString("D:(A;;FRFW;;;BU)")
	if err != nil {
		return err
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return err
	}
	return windows.SetNamedSecurityInfo(settings, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.UNPROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil)
}

// readAutoUpdate keeps the switch an earlier install left: off only when
// the file says false.
func readAutoUpdate(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return true
	}
	defer f.Close()
	b, _ := io.ReadAll(io.LimitReader(f, 4096))
	return !bytes.Contains(b, []byte("false"))
}

func taskName(app setup.App) string { return app.Name + " Update" }

// taskTemplate runs the program with --update once a day and five minutes
// after the computer starts, as SYSTEM, the way browsers keep themselves
// current.
var taskTemplate = template.Must(template.New("task").Funcs(template.FuncMap{"x": xmlText}).Parse(
	`<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Author>{{x .Publisher}}</Author>
    <Description>Installs new versions of {{x .Name}} from its releases on GitHub. Uninstalling {{x .Name}} removes this task.</Description>
  </RegistrationInfo>
  <Triggers>
    <BootTrigger>
      <Delay>PT5M</Delay>
    </BootTrigger>
    <CalendarTrigger>
      <StartBoundary>2000-01-01T12:00:00</StartBoundary>
      <RandomDelay>PT1H</RandomDelay>
      <ScheduleByDay>
        <DaysInterval>1</DaysInterval>
      </ScheduleByDay>
    </CalendarTrigger>
  </Triggers>
  <Principals>
    <Principal id="System">
      <UserId>S-1-5-18</UserId>
      <RunLevel>HighestAvailable</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <StartWhenAvailable>true</StartWhenAvailable>
    <ExecutionTimeLimit>PT1H</ExecutionTimeLimit>
  </Settings>
  <Actions Context="System">
    <Exec>
      <Command>"{{x .Exe}}"</Command>
      <Arguments>--update</Arguments>
    </Exec>
  </Actions>
</Task>
`))

func xmlText(s string) string {
	var b strings.Builder
	xml.EscapeText(&b, []byte(s))
	return b.String()
}

// taskXML renders the task's definition as schtasks reads it: UTF-16 with a
// byte order mark and CRLF line ends.
func taskXML(app setup.App, exe string) []byte {
	var b strings.Builder
	taskTemplate.Execute(&b, struct{ Name, Publisher, Exe string }{app.Name, app.Publisher, exe})
	text := strings.ReplaceAll(b.String(), "\n", "\r\n")
	u := append([]uint16{0xfeff}, utf16.Encode([]rune(text))...)
	out := make([]byte, 2*len(u))
	for i, c := range u {
		out[2*i], out[2*i+1] = byte(c), byte(c>>8)
	}
	return out
}

// createTask registers the update task. The definition is written to a
// folder only administrators can change, since whoever could swap it before
// schtasks reads it would choose what SYSTEM runs.
func (p *platform) createTask(app setup.App, exe string) error {
	tmp, err := privateTemp()
	if err != nil {
		return err
	}
	defer removeTree(tmp)
	def := filepath.Join(tmp, "task.xml")
	if err := os.WriteFile(def, taskXML(app, exe), 0o600); err != nil {
		return err
	}
	return p.schtasks("/Create", "/TN", taskName(app), "/XML", def, "/F")
}

func (p *platform) deleteTask(app setup.App) {
	p.schtasks("/Delete", "/TN", taskName(app), "/F")
}

func runSchtasks(args ...string) error {
	sys, err := windows.GetSystemDirectory()
	if err != nil {
		return err
	}
	out, err := exec.Command(filepath.Join(sys, "schtasks.exe"), args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("schtasks %s: %v: %s", args[0], err, bytes.TrimSpace(out))
	}
	return nil
}

// privateTemp creates an empty folder under the Windows temp folder that only
// administrators can change. The caller removes it.
func privateTemp() (string, error) {
	win, err := windows.GetWindowsDirectory()
	if err != nil {
		return "", err
	}
	var name [8]byte
	rand.Read(name[:])
	dir := filepath.Join(win, "Temp", "pommelporter-"+hex.EncodeToString(name[:]))
	if err := createLocked(dir); err != nil {
		return "", err
	}
	if err := lockDir(dir); err != nil {
		os.Remove(dir)
		return "", err
	}
	return dir, nil
}
