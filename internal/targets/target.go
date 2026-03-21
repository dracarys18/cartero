package targets

import (
	"cartero/internal/components"
	blueskypkg "cartero/internal/targets/bluesky"
	discordpkg "cartero/internal/targets/discord"
	feedpkg "cartero/internal/targets/feed"
	telegrampkg "cartero/internal/targets/telegram"
	"cartero/internal/types"
)

func NewFeedTarget(name string, registry *components.Registry) types.Target {
	return feedpkg.New(name, registry)
}

func NewDiscordTarget(name string, channelID, channelType string, registry *components.Registry) types.Target {
	return discordpkg.New(name, channelID, channelType, registry)
}

func NewBlueskyTarget(name string, languages []string, registry *components.Registry) types.Target {
	return blueskypkg.New(name, languages, registry)
}

func NewTelegramTarget(name string, chatID int64, registry *components.Registry) types.Target {
	return telegrampkg.New(name, chatID, registry)
}
