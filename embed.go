package cartero

import "embed"

//go:embed scripts/scrapers/*.lua
var EmbeddedScripts embed.FS
