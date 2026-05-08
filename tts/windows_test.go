package tts

import (
	"context"
	"runtime"
	"testing"
)

func TestWindowsEngine_ListVoices(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test")
	}

	cfg := EngineConfig{
		SpeechRateMultiplier: 1.0,
	}
	engine := newWindowsEngine(cfg)

	voices, err := engine.ListVoices("en")
	if err != nil {
		t.Fatalf("ListVoices failed: %v", err)
	}

	if len(voices) == 0 {
		t.Log("No voices found, but no error returned. This might be normal on some CI environments, but on a local Windows machine, it should find voices.")
	}
}

func TestWindowsEngine_Speak(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test")
	}

	cfg := EngineConfig{
		SpeechRateMultiplier: 1.0,
	}
	engine := newWindowsEngine(cfg)

	// We don't want to actually make noise in tests if possible, 
	// but SAPI Speak is usually synchronous unless flags are passed.
	// For testing, we just want to ensure it doesn't return an error (like CLSID error).
	err := engine.Speak(context.Background(), "Test", "en")
	if err != nil {
		t.Fatalf("Speak failed: %v", err)
	}
}
