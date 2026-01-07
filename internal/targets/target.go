package targets

import (
	"cartero/internal/components"
	discordpkg "cartero/internal/targets/discord"
	feedpkg "cartero/internal/targets/feed"
	"cartero/internal/types"
)

func NewFeedTarget(name string, registry *components.Registry) types.Target {
	return feedpkg.New(name, registry)
}

func NewDiscordTarget(name string, channelID, channelType string, registry *components.Registry) types.Target {
	return discordpkg.New(name, channelID, channelType, registry)
}
