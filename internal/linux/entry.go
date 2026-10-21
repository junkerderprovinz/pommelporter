//go:build linux

package linux

import (
	"path/filepath"
	"strings"

	"github.com/junkerderprovinz/pommelporter/internal/setup"
)

// desktopEntry is the menu entry, which the desktop copy repeats. Its
// Uninstall action is what the menu offers on a right click.
func desktopEntry(app setup.App, p paths) string {
	categories := app.Categories
	if len(categories) == 0 {
		categories = []string{"Utility"}
	}
	var b strings.Builder
	line := func(key, value string) { b.WriteString(key + "=" + value + "\n") }
	b.WriteString("[Desktop Entry]\n")
	line("Type", "Application")
	line("Version", "1.5")
	line("Name", escapeValue(app.Name))
	line("Exec", execArg(p.exe))
	line("TryExec", escapeValue(p.exe))
	if len(app.Icon) > 0 {
		line("Icon", strings.TrimSuffix(filepath.Base(p.icon), ".png"))
	}
	line("Terminal", "false")
	line("Categories", escapeValue(strings.Join(categories, ";")+";"))
	// The apps hand GTK their Name as the program name, which is the window
	// class X11 docks match against.
	line("StartupWMClass", escapeValue(app.Name))
	line("StartupNotify", "true")
	line("Actions", "uninstall;")
	b.WriteString("\n[Desktop Action uninstall]\n")
	line("Name", "Uninstall")
	line("Exec", execArg(filepath.Join(p.dir, uninstaller))+" --uninstall")
	return b.String()
}

// execArg quotes one argument of an Exec line: the quoting the desktop entry
// spec asks of an argument with reserved characters first, then the escaping
// every string value goes through.
func execArg(arg string) string {
	arg = strings.ReplaceAll(arg, "%", "%%")
	if strings.ContainsAny(arg, " \t\n\"'\\><~|&;$*?#()`") {
		arg = `"` + strings.NewReplacer(`"`, `\"`, "`", "\\`", `$`, `\$`, `\`, `\\`).Replace(arg) + `"`
	}
	return escapeValue(arg)
}

func escapeValue(s string) string {
	return strings.NewReplacer(`\`, `\\`, "\n", `\n`, "\t", `\t`, "\r", `\r`).Replace(s)
}
