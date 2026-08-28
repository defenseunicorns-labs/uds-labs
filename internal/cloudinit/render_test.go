// Copyright 2026 Defense Unicorns
// SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

package cloudinit

import (
	"strings"
	"testing"
	"testing/fstest"
	"text/template"
)

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
