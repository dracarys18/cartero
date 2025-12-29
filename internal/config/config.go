package config

import (
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Bot        BotConfig                  `toml:"bot"`
	Storage    StorageConfig              `toml:"storage"`
	Platforms  map[string]PlatformConfig  `toml:"platforms"`
	Sources    map[string]SourceConfig    `toml:"sources"`
	Processors map[string]ProcessorConfig `toml:"processors"`
	Targets    map[string]TargetConfig    `toml:"targets"`
}

type BotConfig struct {
	Name     string `toml:"name"`
	Interval string `toml:"interval"`
	RunOnce  bool   `toml:"run_once"`
	Sleep    string `toml:"sleep"`
}

type StorageConfig struct {
	Type string `toml:"type"`
	Path string `toml:"path"`
}

type PlatformConfig struct {
	Type     string                 `toml:"type"`
	Sleep    string                 `toml:"sleep"`
	Settings map[string]interface{} `toml:"settings"`
}

type SourceConfig struct {
	Type     string                 `toml:"type"`
	Enabled  bool                   `toml:"enabled"`
	Targets  []string               `toml:"targets"`
	Settings map[string]interface{} `toml:"settings"`
}

type ProcessorConfig struct {
	Type      string                 `toml:"type"`
	Enabled   bool                   `toml:"enabled"`
	DependsOn []string               `toml:"depends_on"`
	Settings  map[string]interface{} `toml:"settings"`
}

type TargetConfig struct {
	Type     string                 `toml:"type"`
	Enabled  bool                   `toml:"enabled"`
	Platform string                 `toml:"platform"`
	Settings map[string]interface{} `toml:"settings"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

func validateConfig(config *Config) error {
	if config.Bot.Name == "" {
		config.Bot.Name = "cartero"
	}

	if config.Bot.Interval == "" {
		config.Bot.Interval = "5m"
	}

	if _, err := time.ParseDuration(config.Bot.Interval); err != nil {
		return fmt.Errorf("invalid interval: %w", err)
	}

	if config.Bot.Sleep == "" {
		config.Bot.Sleep = "2s"
	}

	if _, err := time.ParseDuration(config.Bot.Sleep); err != nil {
		return fmt.Errorf("invalid sleep duration: %w", err)
	}

	if config.Storage.Type == "" {
		config.Storage.Type = "sqlite"
	}

	if config.Storage.Path == "" {
		config.Storage.Path = "./cartero.db"
	}

	enabledSources := 0
	for _, src := range config.Sources {
		if src.Enabled {
			enabledSources++
		}
	}
	if enabledSources == 0 {
		return fmt.Errorf("at least one source must be enabled")
	}

	enabledTargets := 0
	for _, tgt := range config.Targets {
		if tgt.Enabled {
			enabledTargets++
		}
	}
	if enabledTargets == 0 {
		return fmt.Errorf("at least one target must be enabled")
	}

	return nil
}

func GetString(settings map[string]interface{}, key string, defaultValue string) string {
	if val, ok := settings[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

func GetInt(settings map[string]interface{}, key string, defaultValue int) int {
	if val, ok := settings[key]; ok {
		if i, ok := val.(int64); ok {
			return int(i)
		}
		if i, ok := val.(int); ok {
			return i
		}
	}
	return defaultValue
}

func GetBool(settings map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := settings[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultValue
}

func GetStringSlice(settings map[string]interface{}, key string) []string {
	if val, ok := settings[key]; ok {
		if arr, ok := val.([]interface{}); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return []string{}
}

func GetStringMap(settings map[string]interface{}, key string) map[string]string {
	if val, ok := settings[key]; ok {
		if m, ok := val.(map[string]interface{}); ok {
			result := make(map[string]string)
			for k, v := range m {
				if str, ok := v.(string); ok {
					result[k] = str
				}
			}
			return result
		}
	}
	return map[string]string{}
}

func GetDuration(settings map[string]interface{}, key string, defaultValue time.Duration) time.Duration {
	if val, ok := settings[key]; ok {
		if str, ok := val.(string); ok {
			if d, err := time.ParseDuration(str); err == nil {
				return d
			}
		}
	}
	return defaultValue
}
