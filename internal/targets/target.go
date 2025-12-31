package targets

import (
	"cartero/internal/core"
	discordpkg "cartero/internal/targets/discord"
	feedpkg "cartero/internal/targets/feed"
)

func NewFeedTarget(name string) core.Target {
	return feedpkg.New(name)
}

func NewDiscordTarget(name string, channelID, channelType string) core.Target {
	return discordpkg.New(name, channelID, channelType)
}
