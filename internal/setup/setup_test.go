package setup

import (
	"reflect"
	"testing"
)

func TestChoicesRunOnlyWhereTheyApply(t *testing.T) {
	app := App{Choices: []Choice{
		{ID: "folders", Platforms: []string{"windows", "linux"}, Do: []string{"--file-manager", "on"}, Undo: []string{"--file-manager", "off"}},
		{ID: "mac", Platforms: []string{"darwin"}, Do: []string{"--mac"}, Undo: []string{"--no-mac"}},
		{ID: "notes", Do: []string{"--notes"}},
	}}
	if got := app.Ticked("windows", Options{}); got != nil {
		t.Fatalf("nothing ticked, got %v", got)
	}
	ticked := Options{Choices: map[string]bool{"folders": true, "mac": true, "notes": true}}
	if got, want := app.Ticked("windows", ticked), [][]string{{"--file-manager", "on"}, {"--notes"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ticked %v", got)
	}
	if got, want := app.Undos("linux"), [][]string{{"--file-manager", "off"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("undos %v", got)
	}
	if got := len(app.ChoicesOn("windows")); got != 2 {
		t.Fatalf("%d choices on Windows", got)
	}
}
