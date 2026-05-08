package config

import (
	"encoding/json"
	"os"

	"github.com/fsnotify/fsnotify"
)

type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Config struct {
	Comment              string   `json:"_comment"`
	AutoFadeDelay        int      `json:"autoFadeDelay"`
	CloudTTSAPIKey       string   `json:"cloudTTSAPIKey"`
	CloudTTSEnabled      bool     `json:"cloudTTSEnabled"`
	EnglishVoiceName     string   `json:"englishVoiceName"`
	MaxQueueSize         int      `json:"maxQueueSize"`
	OverlayPosition      Position `json:"overlayPosition"`
	PinLastMessageHotkey string   `json:"pinLastMessageHotkey"`
	SpeechRateMultiplier float64  `json:"speechRateMultiplier"`
	ThaiVoiceName        string   `json:"thaiVoiceName"`
	ToggleOverlayHotkey  string   `json:"toggleOverlayHotkey"`
	TwitchChannel        string   `json:"twitchChannel"`
	TwitchOAuthToken     string   `json:"twitchOAuthToken"`
	TTSEngine            string   `json:"ttsEngine"`
}

func Default() *Config {
	return &Config{
		Comment:               "This is the configuration file for Chat Alert. You can edit this file directly, but make sure to follow the JSON format. The app will reload the configuration if you save changes while it is running. Settings are also accessible via the system tray icon.",
		SpeechRateMultiplier:   1.0,
		AutoFadeDelay:          5,
		MaxQueueSize:           20,
		OverlayPosition:        Position{X: 0, Y: 0},
		ToggleOverlayHotkey:    "Ctrl+Shift+T",
		PinLastMessageHotkey:   "Ctrl+Shift+P",
		TTSEngine:             "local",
	}
}

func Load(path string) (*Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, nil
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func Save(path string, cfg *Config) error {
	cfg.Comment = "This is the configuration file for Chat Alert. You can edit this file directly, but make sure to follow the JSON format. The app will reload the configuration if you save changes while it is running. Settings are also accessible via the system tray icon."
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func Watch(path string, onChange func(*Config), done chan struct{}) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					cfg, err := Load(path)
					if err == nil {
						onChange(cfg)
					}
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			case <-done:
				return
			}
		}
	}()

	return watcher.Add(path)
}