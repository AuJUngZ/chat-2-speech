package queue

import (
	"chat-alert/irc"
	"chat-alert/tts"
	"context"
	"sync"
	"time"
)

type Config struct {
	MaxSize       int
	AutoFadeDelay int
}

type LifecycleEmitter interface {
	OnSpeakStart(msg irc.ChatMessage)
	OnSpeakFadeStart(msg irc.ChatMessage)
	OnQueueUpdate(messages []irc.ChatMessage)
}

type MessageQueue struct {
	messages      []irc.ChatMessage
	ttsEngine     tts.Engine
	maxSize       int
	autoFadeDelay int
	emitter       LifecycleEmitter
	quit          chan struct{}

	skip       chan struct{}
	cancel     context.CancelFunc
	mu         sync.Mutex
	cond       *sync.Cond
	paused     bool
}

func New(engine tts.Engine, cfg Config) *MessageQueue {
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 50
	}
	if cfg.AutoFadeDelay <= 0 {
		cfg.AutoFadeDelay = 5
	}
	q := &MessageQueue{
		messages:      make([]irc.ChatMessage, 0, cfg.MaxSize),
		ttsEngine:     engine,
		maxSize:        cfg.MaxSize,
		autoFadeDelay:  cfg.AutoFadeDelay,
		quit:           make(chan struct{}),
		skip:           make(chan struct{}, 1),
	}
	q.cond = sync.NewCond(&q.mu)
	go q.worker()
	return q
}

func (q *MessageQueue) SetEmitter(emitter LifecycleEmitter) {
	q.emitter = emitter
}

func (q *MessageQueue) Stop() {
	q.mu.Lock()
	if q.cancel != nil {
		q.cancel()
	}
	select {
	case <-q.quit:
		// already closed
	default:
		close(q.quit)
	}
	q.cond.Broadcast()
	q.mu.Unlock()
}

func (q *MessageQueue) Pause() {
	q.mu.Lock()
	q.paused = true
	q.mu.Unlock()
}

func (q *MessageQueue) Resume() {
	q.mu.Lock()
	q.paused = false
	q.cond.Broadcast()
	q.mu.Unlock()
}

func (q *MessageQueue) SkipCurrent() {
	q.mu.Lock()
	if q.cancel != nil {
		q.cancel()
	}
	q.mu.Unlock()

	select {
	case q.skip <- struct{}{}:
	default:
	}
}

func (q *MessageQueue) UpdateConfig(cfg Config) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.autoFadeDelay = cfg.AutoFadeDelay
}

func (q *MessageQueue) IsPaused() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.paused
}

func (q *MessageQueue) worker() {
	for {
		q.mu.Lock()
		for len(q.messages) == 0 || q.paused {
			select {
			case <-q.quit:
				q.mu.Unlock()
				return
			default:
			}
			q.cond.Wait()
		}

		msg := q.messages[0]
		q.messages = q.messages[1:]
		if q.emitter != nil {
			q.emitter.OnQueueUpdate(q.messages)
			q.emitter.OnSpeakStart(msg)
		}

		// Prepare context for cancellation
		ctx, cancel := context.WithCancel(context.Background())
		q.cancel = cancel
		q.mu.Unlock()

		// Speak blocks until speech is finished or cancelled
		q.ttsEngine.Speak(ctx, msg.Text, "th")

		q.mu.Lock()
		cancel()
		q.cancel = nil
		for q.paused {
			q.cond.Wait()
			select {
			case <-q.quit:
				q.mu.Unlock()
				return
			default:
			}
		}
		q.mu.Unlock()

		if q.emitter != nil {
			// Wait for the auto-fade delay after speaking, unless skipped
			select {
			case <-time.After(time.Duration(q.autoFadeDelay) * time.Second):
			case <-q.skip:
			case <-q.quit:
				return
			}

			q.emitter.OnSpeakFadeStart(msg)

			select {
			case <-time.After(500 * time.Millisecond):
			case <-q.skip:
			case <-q.quit:
				return
			}
		}

		select {
		case <-q.skip:
		default:
		}
	}
}

func (q *MessageQueue) Enqueue(msg irc.ChatMessage) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.messages) >= q.maxSize {
		q.messages = q.messages[1:]
	}
	q.messages = append(q.messages, msg)

	if q.emitter != nil {
		q.emitter.OnQueueUpdate(q.messages)
	}
	q.cond.Broadcast()
}

func (q *MessageQueue) IsEmpty() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.messages) == 0
}

func (q *MessageQueue) Length() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.messages)
}
