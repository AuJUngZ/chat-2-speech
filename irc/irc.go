package irc

import (
	"crypto/tls"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/thoj/go-ircevent"
)

type Config interface {
	TwitchOAuthToken() string
	TwitchChannel() string
}

type ChatMessage struct {
	Username string
	Text     string
	Channel  string
	Raw      string
	Platform string
}

type Client struct {
	config              Config
	conn                *irc.Connection
	mu                  sync.RWMutex
	connected           bool
	reconnectEnabled    bool
	maxReconnectAttempts int
	reconnectAttempt    int
	callbacks           []func(ChatMessage)
	OnConnected         func()
	OnDisconnected      func(reason string)
	OnReconnecting      func(attempt int, maxAttempts int)
}

func NewClient(cfg Config) *Client {
	return &Client{
		config:              cfg,
		reconnectEnabled:    true,
		maxReconnectAttempts: 5,
	}
}

func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	token := c.config.TwitchOAuthToken()
	channel := c.config.TwitchChannel()
	nick := strings.TrimPrefix(channel, "#")

	conn := irc.IRC(nick, "chat-alert")

	conn.UseTLS = true
	conn.TLSConfig = &tls.Config{
		ServerName: "irc.chat.twitch.tv",
	}

	conn.Password = token

	conn.Debug = false

	conn.AddCallback("PRIVMSG", func(e *irc.Event) {
		c.handlePrivmsg(e)
	})

	conn.AddCallback("DISCONNECTED", func(e *irc.Event) {
		c.handleDisconnect()
	})

	if err := conn.Connect("irc.chat.twitch.tv:6697"); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.conn = conn
	c.connected = true

	if c.OnConnected != nil {
		go c.OnConnected()
	}

	go conn.Loop()

	conn.Join(channel)

	return nil
}

func (c *Client) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.conn.Quit()
		c.connected = false
	}
}

func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

func (c *Client) OnChatMessage(cb func(ChatMessage)) {
	c.callbacks = append(c.callbacks, cb)
}

func (c *Client) handlePrivmsg(e *irc.Event) {
	fmt.Printf("IRC: received PRIVMSG: %v\n", e.Raw)
	if len(e.Arguments) < 2 {
		return
	}

	channel := e.Arguments[0]
	message := e.Arguments[1]

	source := e.Source
	username := strings.Split(source, "!")[0]

	msg := ChatMessage{
		Username: username,
		Text:     message,
		Channel:  channel,
		Raw:      e.Raw,
		Platform: "twitch",
	}

	for _, cb := range c.callbacks {
		go cb(msg)
	}
}

func (c *Client) handleDisconnect() {
	c.mu.Lock()
	c.connected = false
	c.mu.Unlock()

	if c.OnDisconnected != nil {
		go c.OnDisconnected("connection lost")
	}

	if c.reconnectEnabled && c.reconnectAttempt < c.maxReconnectAttempts {
		if c.OnReconnecting != nil {
			go c.OnReconnecting(c.reconnectAttempt+1, c.maxReconnectAttempts)
		}

		delay := c.backoffDelay(c.reconnectAttempt)
		c.reconnectAttempt++

		go func() {
			time.Sleep(delay)
			c.Connect()
		}()
	}
}

func (c *Client) backoffDelay(attempt int) time.Duration {
	base := time.Second
	for i := 0; i < attempt; i++ {
		base *= 2
	}
	return base
}