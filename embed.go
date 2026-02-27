package cartero

import "embed"

//go:embed scrapers/*.lua
var EmbeddedScripts embed.FS
