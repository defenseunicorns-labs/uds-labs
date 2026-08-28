// Copyright 2026 Defense Unicorns
// SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

package labplatform

import "embed"

//go:embed web/static
var StaticFiles embed.FS

//go:embed scenarios
var ScenariosFS embed.FS

//go:embed vm
var VMFiles embed.FS

//go:embed config
var ConfigFiles embed.FS
