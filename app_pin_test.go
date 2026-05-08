package main

import (
	"chat-alert/config"
	"chat-alert/irc"
	"context"
	"testing"
)

type mockTTSEngine struct{}

func (m *mockTTSEngine) Speak(ctx context.Context, text, lang string) error { return nil }
func (m *mockTTSEngine) EstimateDuration(text, lang string) float64 { return 0 }
func (m *mockTTSEngine) ListVoices(lang string) ([]string, error) { return nil, nil }

func TestPinLastMessagePausesQueue(t *testing.T) {
	app := NewApp()
	app.cfg = config.Default()
	app.ttsEngine = &mockTTSEngine{}
	app.initQueue()

	// Add a message to pinned history
	app.OnSpeakStart(irc.ChatMessage{Username: "user", Text: "message"})

	if app.msgQueue.IsPaused() {
		t.Fatal("queue should not be paused initially")
	}

	app.PinLastMessage()

	if !app.msgQueue.IsPaused() {
		t.Error("expected queue to be paused after PinLastMessage")
	}
}

func TestPinOnlyWorksIfMessageActive(t *testing.T) {
	app := NewApp()
	app.cfg = config.Default()
	app.ttsEngine = &mockTTSEngine{}
	app.initQueue()

	// No message active yet
	app.PinLastMessage()
	if app.msgQueue.IsPaused() {
		t.Error("queue should NOT be paused if no message has started speaking")
	}

	// Start speaking
	msg := irc.ChatMessage{Username: "user", Text: "message"}
	app.OnSpeakStart(msg)

	app.PinLastMessage()
	if !app.msgQueue.IsPaused() {
		t.Error("expected queue to be paused after PinLastMessage while speaking")
	}

	// Reset pause
	app.msgQueue.Resume()

	// Message finished/faded
	app.OnSpeakFadeStart(msg)

	app.PinLastMessage()
	if app.msgQueue.IsPaused() {
		t.Error("queue should NOT be paused if message has finished speaking (faded)")
	}
}

func TestUnpinMessageResumesQueue(t *testing.T) {
	app := NewApp()
	app.cfg = config.Default()
	app.ttsEngine = &mockTTSEngine{}
	app.initQueue()

	app.OnSpeakStart(irc.ChatMessage{Username: "user", Text: "message"})
	app.PinLastMessage()

	if !app.msgQueue.IsPaused() {
		t.Fatal("queue should be paused")
	}

	app.UnpinMessage()

	if app.msgQueue.IsPaused() {
		t.Error("expected queue to be resumed after UnpinMessage")
	}
}

func TestPassMessageResumesQueue(t *testing.T) {
	app := NewApp()
	app.cfg = config.Default()
	app.ttsEngine = &mockTTSEngine{}
	app.initQueue()

	app.OnSpeakStart(irc.ChatMessage{Username: "user", Text: "message"})
	app.PinLastMessage()

	if !app.msgQueue.IsPaused() {
		t.Fatal("queue should be paused")
	}

	app.PassMessage()

	if app.msgQueue.IsPaused() {
		t.Error("expected queue to be resumed after PassMessage")
	}
}
