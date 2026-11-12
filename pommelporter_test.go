package pommelporter

import (
	"context"
	"testing"

	"github.com/junkerderprovinz/pommelporter/internal/setup"
)

func TestComponentsThatCannotBeFetchedSafelyAreRefused(t *testing.T) {
	name := map[string]string{"en": "Runtime"}
	fetch := func(context.Context, func(int64, int64)) error { return nil }
	verify := func(string) error { return nil }
	for _, tc := range []struct {
		why string
		c   []setup.Component
	}{
		{"no ID", []setup.Component{{Name: name, Fetch: fetch}}},
		{"the same ID twice", []setup.Component{{ID: "rt", Name: name, Fetch: fetch}, {ID: "rt", Name: name, Fetch: fetch}}},
		{"no name", []setup.Component{{ID: "rt", Fetch: fetch}}},
		{"neither a Fetch nor a URL", []setup.Component{{ID: "rt", Name: name, Target: "/opt/rt"}}},
		{"a URL without a Target", []setup.Component{{ID: "rt", Name: name, URL: "https://example.org/rt", SHA256: "ab"}}},
		{"plain http", []setup.Component{{ID: "rt", Name: name, Target: "/opt/rt", URL: "http://example.org/rt", SHA256: "ab"}}},
		{"nothing that checks the download", []setup.Component{{ID: "rt", Name: name, Target: "/opt/rt", URL: "https://example.org/rt"}}},
	} {
		if err := checkComponents(tc.c); err == nil {
			t.Errorf("%s was accepted", tc.why)
		}
	}

	ok := []setup.Component{
		{ID: "a", Name: name, Fetch: fetch},
		{ID: "b", Name: name, Target: "/opt/b", URL: "https://example.org/b", SHA256: "ab"},
		{ID: "c", Name: name, Target: "/opt/c", URL: "https://example.org/c", Verify: verify},
	}
	if err := checkComponents(ok); err != nil {
		t.Fatalf("refused %v", err)
	}
}
