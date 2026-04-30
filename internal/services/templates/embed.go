package templates

import "embed"

//go:embed default/*.html small/*.html thermal/*.html
var FS embed.FS
