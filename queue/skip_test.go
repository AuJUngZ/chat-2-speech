package queue

import (
	"chat-alert/irc"
	"testing"
	"time"
)

func TestSkipInterruptsSpeak(t *testing.T) {
	m := &mockEngine{speakWait: 500 * time.Millisecond}
	cfg := Config{MaxSize: 10, AutoFadeDelay: 5}
	q := New(m, cfg)

	q.Enqueue(irc.ChatMessage{Text: "long message"})

	// Wait for it to start speaking
	time.Sleep(100 * time.Millisecond)

	start := time.Now()
	q.SkipCurrent()

	// Wait a bit for the worker to process the skip
	time.Sleep(50 * time.Millisecond)

	elapsed := time.Since(start)
	if elapsed > 200*time.Millisecond {
		t.Errorf("expected skip to interrupt immediately, but took %v", elapsed)
	}

	// Message should NOT be in spoken because it was cancelled before finishing
	if len(m.Spoken()) != 0 {
		t.Error("expected 0 spoken messages after skip during Speak")
	}
}

func TestSkipInterruptsFadeDelay(t *testing.T) {
	m := &mockEngine{speakWait: 10 * time.Millisecond}
	emitter := &mockLifecycleEmitter{}
	cfg := Config{MaxSize: 10, AutoFadeDelay: 5}
	q := New(m, cfg)
	q.SetEmitter(emitter)

	q.Enqueue(irc.ChatMessage{Text: "message"})

	// Wait for Speak to finish and enter fade delay
	time.Sleep(100 * time.Millisecond)

	if len(emitter.FadeStartCalls()) != 0 {
		t.Fatal("should still be in fade delay")
	}

	q.SkipCurrent()

	// Wait for worker to process skip
	time.Sleep(50 * time.Millisecond)

	if len(emitter.FadeStartCalls()) != 1 {
		t.Error("expected skip to immediately trigger OnSpeakFadeStart")
	}
}
