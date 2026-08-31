package cloudinit

import (
	"strings"
	"testing"
	"testing/fstest"
	"text/template"
)

func TestRenderStreamsSetupOutputUntilScenarioIsReady(t *testing.T) {
	tmpl := template.Must(template.ParseFiles("../../vm/user-data.sh.gotmpl"))
	scenarios := fstest.MapFS{
		"example/setup.sh": &fstest.MapFile{Data: []byte("#!/bin/bash\n")},
	}

	got, err := Render(tmpl, scenarios, "example", "# inject", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		`tail -n +1 -F "$SETUP_LOG"`,
		`while [ ! -f "$SETUP_DIR/ready" ]`,
		"Live setup output follows below.",
	} {
		if !strings.Contains(got, required) {
			t.Errorf("rendered user-data is missing %q", required)
		}
	}
}

func TestRenderUsesEnvironmentProvidedDNSResolvers(t *testing.T) {
	tmpl := template.Must(template.ParseFiles("../../vm/user-data.sh.gotmpl"))
	scenarios := fstest.MapFS{
		"example/setup.sh": &fstest.MapFile{Data: []byte("#!/bin/bash\n")},
	}

	got, err := Render(tmpl, scenarios, "example", "# inject", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "/run/systemd/resolve/resolv.conf") {
		t.Fatal("guest bootstrap must discover the resolver provided by its environment")
	}
	if strings.Contains(got, "server=1.1.1.1") || strings.Contains(got, "server=8.8.8.8") {
		t.Fatal("guest bootstrap must not bypass the environment resolver with hardcoded public DNS")
	}
}
