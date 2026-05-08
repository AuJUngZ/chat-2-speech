package main

import (
	"chat-alert/config"
	"os"
	"path/filepath"
	"testing"
)

func TestHotkeyBindingsStoredInConfig(t *testing.T) {
	cfg := config.Default()
	if cfg.ToggleOverlayHotkey != "Ctrl+Shift+T" {
		t.Errorf("expected default ToggleOverlayHotkey Ctrl+Shift+T, got %s", cfg.ToggleOverlayHotkey)
	}
	if cfg.PinLastMessageHotkey != "Ctrl+Shift+P" {
		t.Errorf("expected default PinLastMessageHotkey Ctrl+Shift+P, got %s", cfg.PinLastMessageHotkey)
	}
}

func TestHotkeyConfigPersistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chat-alert-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.Default()
	cfg.ToggleOverlayHotkey = "Ctrl+Alt+X"
	cfg.PinLastMessageHotkey = "Ctrl+Alt+Y"

	configPath := filepath.Join(tmpDir, "config.json")
	if err := config.Save(configPath, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.ToggleOverlayHotkey != "Ctrl+Alt+X" {
		t.Errorf("ToggleOverlayHotkey mismatch: got %s, want Ctrl+Alt+X", loaded.ToggleOverlayHotkey)
	}
	if loaded.PinLastMessageHotkey != "Ctrl+Alt+Y" {
		t.Errorf("PinLastMessageHotkey mismatch: got %s, want Ctrl+Alt+Y", loaded.PinLastMessageHotkey)
	}
}

func TestAppWithHotkeyConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chat-alert-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.Default()
	cfg.ToggleOverlayHotkey = "Ctrl+Shift+W"
	cfg.PinLastMessageHotkey = "Ctrl+Shift+K"

	configPath := filepath.Join(tmpDir, "config.json")
	if err := config.Save(configPath, cfg); err != nil {
		t.Fatal(err)
	}

	app := NewAppWithDir(tmpDir)
	loadedCfg := app.GetConfig()

	if loadedCfg.ToggleOverlayHotkey != "Ctrl+Shift+W" {
		t.Errorf("ToggleOverlayHotkey not loaded: got %s, want Ctrl+Shift+W", loadedCfg.ToggleOverlayHotkey)
	}
	if loadedCfg.PinLastMessageHotkey != "Ctrl+Shift+K" {
		t.Errorf("PinLastMessageHotkey not loaded: got %s, want Ctrl+Shift+K", loadedCfg.PinLastMessageHotkey)
	}
}

func TestUpdateHotkeysFromConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chat-alert-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	app := NewAppWithDir(tmpDir)

	app.cfg.ToggleOverlayHotkey = "Ctrl+Shift+R"
	app.cfg.PinLastMessageHotkey = "Ctrl+Shift+M"

	if app.hotkeyManager != nil {
		app.hotkeyManager.UnregisterAll()
	}
}