package queue

import (
	"context"
	"sync"
	"testing"
	"time"

	"chat-alert/irc"
)

type mockLifecycleEmitter struct {
	onSpeakStartCalls  []irc.ChatMessage
	onFadeStartCalls   []irc.ChatMessage
	onQueueUpdateCalls [][]irc.ChatMessage
	mu                 sync.Mutex
}

func (e *mockLifecycleEmitter) OnSpeakStart(msg irc.ChatMessage) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onSpeakStartCalls = append(e.onSpeakStartCalls, msg)
}

func (e *mockLifecycleEmitter) OnSpeakFadeStart(msg irc.ChatMessage) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onFadeStartCalls = append(e.onFadeStartCalls, msg)
}

func (e *mockLifecycleEmitter) OnQueueUpdate(messages []irc.ChatMessage) {
	e.mu.Lock()
	defer e.mu.Unlock()
	// Copy slice to avoid race or modification issues
	msgsCopy := make([]irc.ChatMessage, len(messages))
	copy(msgsCopy, messages)
	e.onQueueUpdateCalls = append(e.onQueueUpdateCalls, msgsCopy)
}

func (e *mockLifecycleEmitter) SpeakStartCalls() []irc.ChatMessage {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]irc.ChatMessage{}, e.onSpeakStartCalls...)
}

func (e *mockLifecycleEmitter) FadeStartCalls() []irc.ChatMessage {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]irc.ChatMessage{}, e.onFadeStartCalls...)
}

func (e *mockLifecycleEmitter) QueueUpdateCalls() [][]irc.ChatMessage {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.onQueueUpdateCalls
}

type mockEngine struct {
	spoken    []string
	speakErr  error
	speakWait time.Duration
	mu        sync.Mutex
}

func (e *mockEngine) Speak(ctx context.Context, text, lang string) error {
	e.mu.Lock()
	speakWait := e.speakWait
	e.mu.Unlock()

	select {
	case <-time.After(speakWait):
		e.mu.Lock()
		e.spoken = append(e.spoken, text)
		e.mu.Unlock()
		return e.speakErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *mockEngine) EstimateDuration(text, lang string) float64 {
	return float64(len(text)) / 100.0
}

func (e *mockEngine) ListVoices(lang string) ([]string, error) {
	return nil, nil
}

func (e *mockEngine) Spoken() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string{}, e.spoken...)
}

func TestEnqueueAndDequeue(t *testing.T) {
	m := &mockEngine{speakWait: 10 * time.Millisecond}
	cfg := Config{MaxSize: 10}
	q := New(m, cfg)

	q.Enqueue(irc.ChatMessage{Username: "user1", Text: "hello"})
	q.Enqueue(irc.ChatMessage{Username: "user2", Text: "world"})

	time.Sleep(50 * time.Millisecond)

	spoken := m.Spoken()
	if len(spoken) != 2 {
		t.Fatalf("expected 2 spoken messages, got %d", len(spoken))
	}
	if spoken[0] != "hello" {
		t.Errorf("expected first spoken 'hello', got %q", spoken[0])
	}
	if spoken[1] != "world" {
		t.Errorf("expected second spoken 'world', got %q", spoken[1])
	}
}

func TestFIFOOrdering(t *testing.T) {
	m := &mockEngine{speakWait: 5 * time.Millisecond}
	cfg := Config{MaxSize: 10}
	q := New(m, cfg)

	for i := 0; i < 5; i++ {
		q.Enqueue(irc.ChatMessage{Text: string(rune('a' + i))})
	}

	time.Sleep(80 * time.Millisecond)

	spoken := m.Spoken()
	expected := []string{"a", "b", "c", "d", "e"}
	for i, exp := range expected {
		if spoken[i] != exp {
			t.Errorf("position %d: expected %q, got %q", i, exp, spoken[i])
		}
	}
}

func TestQueueOverflowDropsOldest(t *testing.T) {
	m := &mockEngine{speakWait: 50 * time.Millisecond}
	cfg := Config{MaxSize: 3}
	q := New(m, cfg)

	for i := 0; i < 5; i++ {
		q.Enqueue(irc.ChatMessage{Text: string(rune('a' + i))})
	}

	time.Sleep(200 * time.Millisecond)

	spoken := m.Spoken()
	if len(spoken) != 3 {
		t.Fatalf("expected 3 spoken (max queue size), got %d", len(spoken))
	}
	if spoken[0] != "c" {
		t.Errorf("expected first spoken 'c' (oldest dropped), got %q", spoken[0])
	}
}

func TestConcurrentEnqueue(t *testing.T) {
	m := &mockEngine{speakWait: 1 * time.Millisecond}
	cfg := Config{MaxSize: 300}
	q := New(m, cfg)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				q.Enqueue(irc.ChatMessage{Text: string(rune('0' + id))})
			}
		}(i)
	}
	wg.Wait()

	time.Sleep(500 * time.Millisecond)

	spoken := m.Spoken()
	if len(spoken) != 200 {
		t.Errorf("expected 200 messages spoken, got %d", len(spoken))
	}
}

func TestIsEmpty(t *testing.T) {
	m := &mockEngine{speakWait: 50 * time.Millisecond}
	cfg := Config{MaxSize: 10}
	q := New(m, cfg)

	if !q.IsEmpty() {
		t.Error("expected empty queue to report IsEmpty=true")
	}

	q.Enqueue(irc.ChatMessage{Text: "hello"})

	if q.IsEmpty() {
		t.Error("expected non-empty queue to report IsEmpty=false")
	}
}

func TestLength(t *testing.T) {
	m := &mockEngine{speakWait: 100 * time.Millisecond}
	cfg := Config{MaxSize: 10}
	q := New(m, cfg)

	if q.Length() != 0 {
		t.Errorf("expected length 0, got %d", q.Length())
	}

	q.Enqueue(irc.ChatMessage{Text: "a"})
	q.Enqueue(irc.ChatMessage{Text: "b"})
	q.Enqueue(irc.ChatMessage{Text: "c"})

	if q.Length() != 3 {
		t.Errorf("expected length 3, got %d", q.Length())
	}
}

func TestDefaultMaxSize(t *testing.T) {
	m := &mockEngine{speakWait: 50 * time.Millisecond}
	cfg := Config{}
	q := New(m, cfg)

	for i := 0; i < 60; i++ {
		q.Enqueue(irc.ChatMessage{Text: string(rune('a' + i))})
	}

	time.Sleep(3000 * time.Millisecond)

	spoken := m.Spoken()
	if len(spoken) != 50 && len(spoken) != 51 {
		t.Errorf("expected 50 or 51 messages spoken (default max), got %d", len(spoken))
	}
}

func TestOnSpeakStartCalled(t *testing.T) {
	m := &mockEngine{speakWait: 10 * time.Millisecond}
	emitter := &mockLifecycleEmitter{}
	cfg := Config{MaxSize: 10, AutoFadeDelay: 5}
	q := New(m, cfg)
	q.SetEmitter(emitter)

	q.Enqueue(irc.ChatMessage{Username: "user1", Text: "hello"})
	time.Sleep(50 * time.Millisecond)

	calls := emitter.SpeakStartCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 OnSpeakStart call, got %d", len(calls))
	}
	if calls[0].Username != "user1" {
		t.Errorf("expected username 'user1', got %q", calls[0].Username)
	}
	if calls[0].Text != "hello" {
		t.Errorf("expected text 'hello', got %q", calls[0].Text)
	}
}

func TestOnSpeakFadeStartCalled(t *testing.T) {
	m := &mockEngine{speakWait: 10 * time.Millisecond}
	emitter := &mockLifecycleEmitter{}
	cfg := Config{MaxSize: 10, AutoFadeDelay: 1}
	q := New(m, cfg)
	q.SetEmitter(emitter)

	q.Enqueue(irc.ChatMessage{Username: "user1", Text: "hello"})
	waitDuration := time.Duration((float64(len("hello"))/100.0 + float64(cfg.AutoFadeDelay)) * float64(time.Second))
	time.Sleep(waitDuration + 100*time.Millisecond)

	calls := emitter.FadeStartCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 OnSpeakFadeStart call, got %d", len(calls))
	}
	if calls[0].Username != "user1" {
		t.Errorf("expected username 'user1', got %q", calls[0].Username)
	}
}

func TestOnSpeakFadeStartAfterDuration(t *testing.T) {
	m := &mockEngine{speakWait: 10 * time.Millisecond}
	emitter := &mockLifecycleEmitter{}
	cfg := Config{MaxSize: 10, AutoFadeDelay: 1}
	q := New(m, cfg)
	q.SetEmitter(emitter)

	q.Enqueue(irc.ChatMessage{Username: "user1", Text: "hello"})
	time.Sleep(50 * time.Millisecond)

	beforeFade := emitter.FadeStartCalls()
	if len(beforeFade) != 0 {
		t.Errorf("expected no OnSpeakFadeStart calls before delay, got %d", len(beforeFade))
	}

	waitDuration := time.Duration(float64(time.Second)*(0.05+float64(cfg.AutoFadeDelay))) + 200*time.Millisecond
	time.Sleep(waitDuration)

	afterFade := emitter.FadeStartCalls()
	if len(afterFade) != 1 {
		t.Fatalf("expected 1 OnSpeakFadeStart call after delay, got %d", len(afterFade))
	}
}

func TestOnQueueUpdate(t *testing.T) {
	m := &mockEngine{speakWait: 100 * time.Millisecond}
	emitter := &mockLifecycleEmitter{}
	cfg := Config{MaxSize: 10}
	q := New(m, cfg)
	q.SetEmitter(emitter)

	q.Enqueue(irc.ChatMessage{Text: "msg1"})
	q.Enqueue(irc.ChatMessage{Text: "msg2"})

	time.Sleep(50 * time.Millisecond) // Worker starts processing msg1, pops it

	calls := emitter.QueueUpdateCalls()
	// Call 1: msg1 added (queue: [msg1])
	// Call 2: msg2 added (queue: [msg1, msg2])
	// Call 3: worker pops msg1 (queue: [msg2])
	
	foundMsg2Only := false
	for _, c := range calls {
		if len(c) == 1 && c[0].Text == "msg2" {
			foundMsg2Only = true
			break
		}
	}
	
	if !foundMsg2Only {
		t.Errorf("expected a queue update call with only 'msg2' after worker pops 'msg1'")
	}
}

