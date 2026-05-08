package hotkey

import (
	"context"
	"testing"
	"time"
)

func TestParseHotkey(t *testing.T) {
	tests := []struct {
		name    string
		hotkey  string
		wantMod uint32
		wantKey uint16
		wantErr bool
	}{
		{"Ctrl+Shift+T", "Ctrl+Shift+T", 0x0006, 0x54, false},
		{"Ctrl+Shift+P", "Ctrl+Shift+P", 0x0006, 0x50, false},
		{"Ctrl+Alt+A", "Ctrl+Alt+A", 0x0003, 0x41, false},
		{"Single key", "A", 0, 0, true},
		{"Unknown modifier", "Ctrl+Unknown", 0, 0, true},
		{"Unknown key", "Ctrl+Xyz", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mod, key, err := parseHotkey(tt.hotkey)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseHotkey(%q) error = %v, wantErr %v", tt.hotkey, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if mod != tt.wantMod {
					t.Errorf("parseHotkey(%q) mod = 0x%X, want 0x%X", tt.hotkey, mod, tt.wantMod)
				}
				if key != tt.wantKey {
					t.Errorf("parseHotkey(%q) key = 0x%X, want 0x%X", tt.hotkey, key, tt.wantKey)
				}
			}
		})
	}
}

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if m.active.Load() != 0 {
		t.Error("manager should not be active initially")
	}
	if len(m.callbacks) != 0 {
		t.Error("callbacks should be empty")
	}
	if len(m.registered) != 0 {
		t.Error("registered should be empty")
	}
}

func TestManagerIsRegistered(t *testing.T) {
	m := NewManager()
	if m.IsRegistered("Ctrl+Shift+T") {
		t.Error("unregistered hotkey should return false")
	}
}

func TestManagerStartStop(t *testing.T) {
	m := NewManager()
	ctx, cancel := context.WithCancel(context.Background())
	m.Start(ctx)

	if !m.IsActive() {
		t.Error("expected manager to be active after Start")
	}

	cancel()
	// Give it a moment to stop
	for i := 0; i < 10; i++ {
		if !m.IsActive() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if m.IsActive() {
		t.Error("expected manager to be inactive after context cancellation")
	}
}

func TestManagerRegister(t *testing.T) {
	m := NewManager()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)

	// This might fail if the hotkey is already taken, but it should at least return
	err := m.Register(ctx, "Ctrl+Shift+F12", func() {})
	if err != nil {
		t.Logf("Register failed (as expected if hotkey taken or in CI): %v", err)
	} else {
		if !m.IsRegistered("Ctrl+Shift+F12") {
			t.Error("expected hotkey to be registered")
		}
		m.Unregister("Ctrl+Shift+F12")
		if m.IsRegistered("Ctrl+Shift+F12") {
			t.Error("expected hotkey to be unregistered")
		}
	}
}

func TestManagerUnregisterAll(t *testing.T) {
	m := NewManager()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)

	m.Register(ctx, "Ctrl+Shift+F11", func() {})
	m.UnregisterAll()

	if len(m.GetRegisteredHotkeys()) != 0 {
		t.Error("expected all hotkeys to be unregistered")
	}
}