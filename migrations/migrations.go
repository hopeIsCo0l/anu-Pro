// Package migrations embeds all SQL migration files so the migrate binary
// is fully self-contained — no external migration directory needed at runtime.
package migrations

import "embed"

//go:embed public tenant
var FS embed.FS
