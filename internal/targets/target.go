package targets

import (
	"cartero/internal/components"
	"cartero/internal/core"
	discordpkg "cartero/internal/targets/discord"
	feedpkg "cartero/internal/targets/feed"
)

func NewFeedTarget(name string, registry *components.Registry) core.Target {
	return feedpkg.New(name, registry)
}

func NewDiscordTarget(name string, channelID, channelType string, registry *components.Registry) core.Target {
	return discordpkg.New(name, channelID, channelType, registry)
}
