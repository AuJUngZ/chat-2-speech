package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHotkeyBindingsInConfig(t *testing.T) {
	cfg := Default()
	if cfg.ToggleOverlayHotkey != "Ctrl+Shift+T" {
		t.Errorf("ToggleOverlayHotkey = %q; want Ctrl+Shift+T", cfg.ToggleOverlayHotkey)
	}
	if cfg.PinLastMessageHotkey != "Ctrl+Shift+P" {
		t.Errorf("PinLastMessageHotkey = %q; want Ctrl+Shift+P", cfg.PinLastMessageHotkey)
	}
}

func TestConfigStructHasAllFields(t *testing.T) {
	c := Default()
	if c.TwitchOAuthToken != "" {
		t.Errorf("TwitchOAuthToken should have default empty string, got %q", c.TwitchOAuthToken)
	}
	if c.ThaiVoiceName != "" {
		t.Errorf("ThaiVoiceName should have default empty string, got %q", c.ThaiVoiceName)
	}
	if c.EnglishVoiceName != "" {
		t.Errorf("EnglishVoiceName should have default empty string, got %q", c.EnglishVoiceName)
	}
	if c.SpeechRateMultiplier <= 0 {
		t.Errorf("SpeechRateMultiplier should have default > 0, got %v", c.SpeechRateMultiplier)
	}
	if c.AutoFadeDelay <= 0 {
		t.Errorf("AutoFadeDelay should have default > 0, got %v", c.AutoFadeDelay)
	}
	if c.MaxQueueSize <= 0 {
		t.Errorf("MaxQueueSize should have default > 0, got %v", c.MaxQueueSize)
	}
	if c.OverlayPosition.X != 0 || c.OverlayPosition.Y != 0 {
		t.Errorf("OverlayPosition should default to 0,0, got %v", c.OverlayPosition)
	}
	if c.CloudTTSAPIKey != "" {
		t.Errorf("CloudTTSAPIKey should have default empty string, got %q", c.CloudTTSAPIKey)
	}
	if c.ToggleOverlayHotkey != "Ctrl+Shift+T" {
		t.Errorf("ToggleOverlayHotkey should default to Ctrl+Shift+T, got %q", c.ToggleOverlayHotkey)
	}
	if c.PinLastMessageHotkey != "Ctrl+Shift+P" {
		t.Errorf("PinLastMessageHotkey should default to Ctrl+Shift+P, got %q", c.PinLastMessageHotkey)
	}
}

func TestLoadReturnsDefaultsForMissingKeys(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	_, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load should not error on missing file, got error: %v", err)
	}
}

func TestSaveWritesJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := Default()
	cfg.TwitchOAuthToken = "oauth:abc123"
	cfg.ThaiVoiceName = "th-voice"
	cfg.EnglishVoiceName = "en-voice"
	cfg.SpeechRateMultiplier = 1.5
	cfg.AutoFadeDelay = 10
	cfg.MaxQueueSize = 50
	cfg.OverlayPosition = Position{X: 100, Y: 200}
	cfg.CloudTTSAPIKey = "secret-key"

	err := Save(configPath, cfg)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("Save did not create config file at %s", configPath)
	}
}

func TestSaveIncludesCommentHeader(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := Default()
	err := Save(configPath, cfg)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Failed to unmarshal saved JSON: %v", err)
	}

	if _, ok := raw["_comment"]; !ok {
		t.Error("Saved JSON missing '_comment' field")
	}
}

func TestLoadPopulatesDefaultsForMissingFields(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Save minimal config
	data := []byte(`{"twitchChannel": "testchannel"}`)
	err := os.WriteFile(configPath, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write minimal config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.TwitchChannel != "testchannel" {
		t.Errorf("TwitchChannel = %q; want testchannel", cfg.TwitchChannel)
	}

	// Check a default field that was missing in JSON
	if cfg.ToggleOverlayHotkey != "Ctrl+Shift+T" {
		t.Errorf("ToggleOverlayHotkey = %q; want default Ctrl+Shift+T", cfg.ToggleOverlayHotkey)
	}
}

func TestWatchDetectsChanges(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := Default()
	err := Save(configPath, cfg)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	changeCh := make(chan *Config, 1)
	done := make(chan struct{})
	defer close(done)

	err = Watch(configPath, func(newCfg *Config) {
		changeCh <- newCfg
	}, done)
	if err != nil {
		t.Fatalf("Watch failed to start: %v", err)
	}

	// Modify file
	cfg.TwitchChannel = "newchannel"
	err = Save(configPath, cfg)
	if err != nil {
		t.Fatalf("Failed to modify config: %v", err)
	}

	select {
	case updated := <-changeCh:
		if updated.TwitchChannel != "newchannel" {
			t.Errorf("Updated config has TwitchChannel = %q; want newchannel", updated.TwitchChannel)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for config change notification")
	}
}

func TestJSONRoundTripPreservesAllFields(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	original := Default()
	original.TwitchOAuthToken = "oauth:abc123"
	original.ThaiVoiceName = "th-voice"
	original.EnglishVoiceName = "en-voice"
	original.SpeechRateMultiplier = 1.5
	original.AutoFadeDelay = 10
	original.MaxQueueSize = 50
	original.OverlayPosition = Position{X: 100, Y: 200}
	original.CloudTTSAPIKey = "secret-key"
	original.ToggleOverlayHotkey = "Ctrl+Shift+T"
	original.PinLastMessageHotkey = "Ctrl+Shift+P"

	err := Save(configPath, original)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed after Save: %v", err)
	}

	if loaded.TwitchOAuthToken != original.TwitchOAuthToken {
		t.Errorf("TwitchOAuthToken mismatch: got %q, want %q", loaded.TwitchOAuthToken, original.TwitchOAuthToken)
	}
	if loaded.ThaiVoiceName != original.ThaiVoiceName {
		t.Errorf("ThaiVoiceName mismatch: got %q, want %q", loaded.ThaiVoiceName, original.ThaiVoiceName)
	}
	if loaded.EnglishVoiceName != original.EnglishVoiceName {
		t.Errorf("EnglishVoiceName mismatch: got %q, want %q", loaded.EnglishVoiceName, original.EnglishVoiceName)
	}
	if loaded.SpeechRateMultiplier != original.SpeechRateMultiplier {
		t.Errorf("SpeechRateMultiplier mismatch: got %v, want %v", loaded.SpeechRateMultiplier, original.SpeechRateMultiplier)
	}
	if loaded.AutoFadeDelay != original.AutoFadeDelay {
		t.Errorf("AutoFadeDelay mismatch: got %v, want %v", loaded.AutoFadeDelay, original.AutoFadeDelay)
	}
	if loaded.MaxQueueSize != original.MaxQueueSize {
		t.Errorf("MaxQueueSize mismatch: got %v, want %v", loaded.MaxQueueSize, original.MaxQueueSize)
	}
	if loaded.OverlayPosition.X != original.OverlayPosition.X || loaded.OverlayPosition.Y != original.OverlayPosition.Y {
		t.Errorf("OverlayPosition mismatch: got %v, want %v", loaded.OverlayPosition, original.OverlayPosition)
	}
	if loaded.CloudTTSAPIKey != original.CloudTTSAPIKey {
		t.Errorf("CloudTTSAPIKey mismatch: got %q, want %q", loaded.CloudTTSAPIKey, original.CloudTTSAPIKey)
	}
	if loaded.ToggleOverlayHotkey != original.ToggleOverlayHotkey {
		t.Errorf("ToggleOverlayHotkey mismatch: got %q, want %q", loaded.ToggleOverlayHotkey, original.ToggleOverlayHotkey)
	}
	if loaded.PinLastMessageHotkey != original.PinLastMessageHotkey {
		t.Errorf("PinLastMessageHotkey mismatch: got %q, want %q", loaded.PinLastMessageHotkey, original.PinLastMessageHotkey)
	}
}