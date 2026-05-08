package main

import (
	"chat-alert/config"
	"chat-alert/irc"
	"os"
	"path/filepath"
	"testing"
)


func TestAppDataDir(t *testing.T) {
	app := NewApp()
	app.initDirectories()

	dir := app.GetAppDataDir()
	if dir == "" {
		t.Error("AppDataDir returned empty string")
	}

	want := filepath.Join("chat-alert")
	if filepath.Base(dir) != want {
		t.Errorf("AppDataDir = %s; want *%s", dir, want)
	}
}

func TestOverlayWindow(t *testing.T) {
	app := NewApp()
	// Removed startup(ctx) as it's not needed for this check
	if app.IsOverlay() {
		t.Error("overlay should be hidden by default")
	}
}

func TestLogDir(t *testing.T) {
	app := NewApp()
	app.initDirectories()

	dir := app.GetLogDir()
	if dir == "" {
		t.Error("GetLogDir returned empty string")
	}

	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		t.Errorf("log dir %s does not exist", dir)
	} else if err != nil {
		t.Errorf("error statting log dir: %v", err)
	} else if !info.IsDir() {
		t.Errorf("log dir %s is not a directory", dir)
	}
}

func TestSavePosition(t *testing.T) {
	app := NewApp()
	// Use a temporary directory for the test to avoid messing with user config
	tmpDir, err := os.MkdirTemp("", "chat-alert-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	app.appDataDir = tmpDir

	err = app.SavePosition(150, 250)
	if err != nil {
		t.Fatalf("SavePosition failed: %v", err)
	}

	if app.cfg.OverlayPosition.X != 150 || app.cfg.OverlayPosition.Y != 250 {
		t.Errorf("expected position 150, 250, got %d, %d", app.cfg.OverlayPosition.X, app.cfg.OverlayPosition.Y)
	}

	cfg, err := config.Load(filepath.Join(app.appDataDir, "config.json"))
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}
	if cfg.OverlayPosition.X != 150 || cfg.OverlayPosition.Y != 250 {
		t.Errorf("saved file has wrong position: %+v", cfg.OverlayPosition)
	}
}

	func TestRestorePosition(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chat-alert-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.Default()
	cfg.OverlayPosition = config.Position{X: 300, Y: 400}
	err = config.Save(filepath.Join(tmpDir, "config.json"), cfg)
	if err != nil {
		t.Fatal(err)
	}

	app := NewAppWithDir(tmpDir)
	if cfg.OverlayPosition.X != 300 || cfg.OverlayPosition.Y != 400 {
		t.Errorf("expected restored position 300, 400, got %d, %d", app.cfg.OverlayPosition.X, app.cfg.OverlayPosition.Y)
	}
	}

	func TestGetConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chat-alert-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	app := NewAppWithDir(tmpDir)
	cfg := app.GetConfig()
	if cfg == nil {
		t.Fatal("GetConfig returned nil")
	}
	if cfg.SpeechRateMultiplier != 1.0 {
		t.Errorf("expected default speech rate 1.0, got %f", cfg.SpeechRateMultiplier)
	}
	if cfg.ToggleOverlayHotkey != "Ctrl+Shift+T" {
		t.Errorf("expected default ToggleOverlayHotkey Ctrl+Shift+T, got %s", cfg.ToggleOverlayHotkey)
	}
	if cfg.PinLastMessageHotkey != "Ctrl+Shift+P" {
		t.Errorf("expected default PinLastMessageHotkey Ctrl+Shift+P, got %s", cfg.PinLastMessageHotkey)
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chat-alert-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	app := NewAppWithDir(tmpDir)
	cfg := app.GetConfig()
	cfg.TwitchChannel = "testchannel"
	cfg.ToggleOverlayHotkey = "Ctrl+Alt+T"
	cfg.PinLastMessageHotkey = "Ctrl+Alt+P"

	err = app.SaveConfig(cfg)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	loaded, err := config.Load(filepath.Join(tmpDir, "config.json"))
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}
	if loaded.TwitchChannel != "testchannel" {
		t.Errorf("expected TwitchChannel 'testchannel', got '%s'", loaded.TwitchChannel)
	}
	if loaded.ToggleOverlayHotkey != "Ctrl+Alt+T" {
		t.Errorf("expected ToggleOverlayHotkey 'Ctrl+Alt+T', got '%s'", loaded.ToggleOverlayHotkey)
	}
	if loaded.PinLastMessageHotkey != "Ctrl+Alt+P" {
		t.Errorf("expected PinLastMessageHotkey 'Ctrl+Alt+P', got '%s'", loaded.PinLastMessageHotkey)
	}
}

func TestGetPinnedMessages(t *testing.T) {
	app := NewApp()
	app.OnSpeakStart(irc.ChatMessage{Username: "User1", Text: "Hello", Platform: "twitch"})
	msgs := app.GetPinnedMessages()
	if len(msgs) != 1 {
		t.Errorf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Username != "User1" || msgs[0].Text != "Hello" || msgs[0].Platform != "twitch" {
		t.Errorf("unexpected message: %+v", msgs[0])
	}
}

func TestSettingsModeTransitions(t *testing.T) {
	app := NewApp()
	
	// Test EnterSettingsMode
	app.EnterSettingsMode()
	if !app.settingsMode {
		t.Error("expected settingsMode to be true after EnterSettingsMode")
	}

	// Test ExitSettingsMode
	app.ExitSettingsMode()
	if app.settingsMode {
		t.Error("expected settingsMode to be false after ExitSettingsMode")
	}

	// Test CancelSettings
	app.EnterSettingsMode()
	app.CancelSettings()
	if app.settingsMode {
		t.Error("expected settingsMode to be false after CancelSettings")
	}
}