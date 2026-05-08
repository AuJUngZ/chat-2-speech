package irc

import (
	"testing"
	"time"

	"github.com/thoj/go-ircevent"
)

type testConfig struct {
	oauthToken string
	channel    string
}

func (c *testConfig) TwitchOAuthToken() string { return c.oauthToken }
func (c *testConfig) TwitchChannel() string   { return c.channel }

func TestClientInterface(t *testing.T) {
	cfg := &testConfig{
		oauthToken: "oauth:testtoken",
		channel:    "#testuser",
	}

	client := NewClient(cfg)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if !client.reconnectEnabled {
		t.Error("reconnect should be enabled by default")
	}

	if client.maxReconnectAttempts != 5 {
		t.Errorf("maxReconnectAttempts = %d; want %d", client.maxReconnectAttempts, 5)
	}

	if client.IsConnected() {
		t.Error("IsConnected() should be false before Connect()")
	}
}

func TestDisconnect(t *testing.T) {
	cfg := &testConfig{
		oauthToken: "oauth:testtoken",
		channel:    "#testuser",
	}

	client := NewClient(cfg)
	client.Connect()
	client.Disconnect()

	if client.IsConnected() {
		t.Error("IsConnected() should be false after Disconnect()")
	}
}

func TestParsePrivmsg(t *testing.T) {
	cfg := &testConfig{
		oauthToken: "oauth:testtoken",
		channel:    "#testuser",
	}

	client := NewClient(cfg)

	var received []ChatMessage
	client.OnChatMessage(func(msg ChatMessage) {
		received = append(received, msg)
	})

	conn := client.conn
	if conn == nil {
		t.Skip("conn not initialized - integration test")
	}

	conn.RunCallbacks(&irc.Event{
		Code:      "PRIVMSG",
		Source:    "testuser!testuser@test.twitch.tv",
		Arguments: []string{"#testuser", "hello world"},
	})

	if len(received) == 0 {
		t.Fatal("callback not called after PRIVMSG")
	}

	if received[0].Username != "testuser" {
		t.Errorf("Username = %q; want %q", received[0].Username, "testuser")
	}
	if received[0].Text != "hello world" {
		t.Errorf("Text = %q; want %q", received[0].Text, "hello world")
	}
	if received[0].Platform != "twitch" {
		t.Errorf("Platform = %q; want %q", received[0].Platform, "twitch")
	}
}

func TestReconnectsOnDisconnect(t *testing.T) {
	cfg := &testConfig{
		oauthToken: "oauth:testtoken",
		channel:    "#testuser",
	}

	client := NewClient(cfg)

	if !client.reconnectEnabled {
		t.Error("reconnect should be enabled by default")
	}

	if client.maxReconnectAttempts != 5 {
		t.Errorf("maxReconnectAttempts = %d; want %d", client.maxReconnectAttempts, 5)
	}
}

func TestReconnectBackoff(t *testing.T) {
	cfg := &testConfig{
		oauthToken: "oauth:testtoken",
		channel:    "#testuser",
	}

	client := NewClient(cfg)

	delays := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second}
	for i, want := range delays {
		got := client.backoffDelay(i)
		if got != want {
			t.Errorf("backoffDelay(%d) = %v; want %v", i, got, want)
		}
	}
}

func TestOnConnectedCallback(t *testing.T) {
	cfg := &testConfig{
		oauthToken: "oauth:testtoken",
		channel:    "#testuser",
	}

	client := NewClient(cfg)

	connected := false
	client.OnConnected = func() {
		connected = true
	}

	conn := client.conn
	if conn == nil {
		t.Skip("conn not initialized - integration test")
	}

	conn.RunCallbacks(&irc.Event{Code: "CONNECTED"})

	if !connected {
		t.Error("OnConnected callback not fired")
	}
}

func TestOnDisconnectedCallback(t *testing.T) {
	cfg := &testConfig{
		oauthToken: "oauth:testtoken",
		channel:    "#testuser",
	}

	client := NewClient(cfg)

	var receivedReason string
	client.OnDisconnected = func(reason string) {
		receivedReason = reason
	}

	conn := client.conn
	if conn == nil {
		t.Skip("conn not initialized - integration test")
	}

	conn.RunCallbacks(&irc.Event{Code: "DISCONNECTED"})

	if receivedReason == "" {
		t.Error("OnDisconnected callback not fired")
	}
}

func TestOnReconnectingCallback(t *testing.T) {
	cfg := &testConfig{
		oauthToken: "oauth:testtoken",
		channel:    "#testuser",
	}

	client := NewClient(cfg)

	var receivedAttempt, receivedMaxAttempts int
	client.OnReconnecting = func(attempt int, maxAttempts int) {
		receivedAttempt = attempt
		receivedMaxAttempts = maxAttempts
	}

	conn := client.conn
	if conn == nil {
		t.Skip("conn not initialized - integration test")
	}

	conn.RunCallbacks(&irc.Event{Code: "DISCONNECTED"})

	if receivedAttempt == 0 && receivedMaxAttempts == 0 {
		t.Error("OnReconnecting callback not fired")
	}
}