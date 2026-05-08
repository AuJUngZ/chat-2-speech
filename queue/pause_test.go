package queue

import (
	"chat-alert/irc"
	"testing"
	"time"
)

func TestPausePreventsSpeaking(t *testing.T) {
	m := &mockEngine{speakWait: 10 * time.Millisecond}
	cfg := Config{MaxSize: 10}
	q := New(m, cfg)

	q.Pause()
	q.Enqueue(irc.ChatMessage{Text: "paused message"})

	time.Sleep(50 * time.Millisecond)

	spoken := m.Spoken()
	if len(spoken) != 0 {
		t.Errorf("expected 0 spoken messages while paused, got %d", len(spoken))
	}
}

func TestResumeContinuesSpeaking(t *testing.T) {
	m := &mockEngine{speakWait: 10 * time.Millisecond}
	cfg := Config{MaxSize: 10}
	q := New(m, cfg)

	q.Pause()
	q.Enqueue(irc.ChatMessage{Text: "message 1"})
	
	time.Sleep(50 * time.Millisecond)
	if len(m.Spoken()) != 0 {
		t.Fatal("should not have spoken while paused")
	}

	q.Resume()
	
	time.Sleep(50 * time.Millisecond)
	spoken := m.Spoken()
	if len(spoken) != 1 {
		t.Errorf("expected 1 spoken message after resume, got %d", len(spoken))
	}
	if spoken[0] != "message 1" {
		t.Errorf("expected 'message 1', got %q", spoken[0])
	}
}

func TestPauseFinishesCurrentMessage(t *testing.T) {
	// Set speakWait long enough to call Pause while it's speaking
	m := &mockEngine{speakWait: 100 * time.Millisecond}
	cfg := Config{MaxSize: 10}
	q := New(m, cfg)

	q.Enqueue(irc.ChatMessage{Text: "message 1"})
	q.Enqueue(irc.ChatMessage{Text: "message 2"})

	// Wait a bit for message 1 to start
	time.Sleep(20 * time.Millisecond)
	
	q.Pause()

	// Wait long enough for message 1 to finish but message 2 should be blocked
	time.Sleep(200 * time.Millisecond)

	spoken := m.Spoken()
	if len(spoken) != 1 {
		t.Errorf("expected 1 spoken message (message 1 should finish), got %d", len(spoken))
	}
	if spoken[0] != "message 1" {
		t.Errorf("expected 'message 1', got %q", spoken[0])
	}

	q.Resume()
	time.Sleep(150 * time.Millisecond)

	spoken = m.Spoken()
	if len(spoken) != 2 {
		t.Errorf("expected 2 spoken messages after resume, got %d", len(spoken))
	}
	if spoken[1] != "message 2" {
		t.Errorf("expected 'message 2', got %q", spoken[1])
	}
}

func TestStopWhilePaused(t *testing.T) {
	m := &mockEngine{speakWait: 10 * time.Millisecond}
	cfg := Config{MaxSize: 10}
	q := New(m, cfg)

	q.Pause()
	
	done := make(chan struct{})
	go func() {
		q.Stop()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(500 * time.Millisecond):
		t.Error("Stop() blocked while queue was paused")
	}
}

func TestPauseBeforeFadeOut(t *testing.T) {
	m := &mockEngine{speakWait: 100 * time.Millisecond}
	emitter := &mockLifecycleEmitter{}
	cfg := Config{MaxSize: 10, AutoFadeDelay: 1} // 1 second delay
	q := New(m, cfg)
	q.SetEmitter(emitter)

	q.Enqueue(irc.ChatMessage{Text: "message 1"})

	// Wait for Speak to start but not finish
	time.Sleep(50 * time.Millisecond)
	
	// Now Pause it. It should finish Speak, then wait at the lock.
	q.Pause()

	// Wait longer than AutoFadeDelay + remaining Speak time
	time.Sleep(1200 * time.Millisecond)

	if len(emitter.FadeStartCalls()) != 0 {
		t.Error("expected 0 OnSpeakFadeStart calls while paused during delay")
	}

	q.Resume()

	// Wait for AutoFadeDelay to finish after resume (at least 1s)
	time.Sleep(1200 * time.Millisecond)

	if len(emitter.FadeStartCalls()) != 1 {
		t.Errorf("expected 1 OnSpeakFadeStart call after resume, got %d", len(emitter.FadeStartCalls()))
	}
}
