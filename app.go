package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"chat-alert/config"
	"chat-alert/hotkey"
	"chat-alert/irc"
	"chat-alert/queue"
	"chat-alert/tts"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx            context.Context
	overlayVisible bool
	settingsMode   bool
	appDataDir     string
	logDir         string
	cfg            *config.Config
	ircClient      *ircClient
	ttsEngine      tts.Engine
	msgQueue       *queue.MessageQueue
	hotkeyManager  *hotkey.HotkeyManager
	pinnedMessages []irc.ChatMessage
	speakingActive bool
	mu             sync.RWMutex
}

type ircClient struct {
	client *irc.Client
	mu     sync.RWMutex
}

func (a *App) OnSpeakStart(msg irc.ChatMessage) {
	a.mu.Lock()
	a.speakingActive = true
	a.pinnedMessages = append(a.pinnedMessages, msg)
	if len(a.pinnedMessages) > 20 {
		a.pinnedMessages = a.pinnedMessages[1:]
	}
	a.mu.Unlock()
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "tts-started", map[string]string{
			"username": msg.Username,
			"message":  msg.Text,
			"platform": msg.Platform,
		})
	}
}

func (a *App) OnSpeakFadeStart(msg irc.ChatMessage) {
	a.mu.Lock()
	a.speakingActive = false
	a.mu.Unlock()
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "tts-fade-start", nil)
	}
}

func (a *App) OnQueueUpdate(messages []irc.ChatMessage) {
	if a.ctx != nil {
		var queueData []map[string]string
		limit := 4
		if len(messages) < limit {
			limit = len(messages)
		}
		for i := 0; i < limit; i++ {
			msg := messages[i]
			queueData = append(queueData, map[string]string{
				"username": msg.Username,
				"message":  msg.Text,
				"platform": msg.Platform,
			})
		}
		runtime.EventsEmit(a.ctx, "queue-updated", queueData)
	}
}

func (a *App) OnTTSError(msg irc.ChatMessage, err error) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "tts-error", map[string]interface{}{
			"username": msg.Username,
			"message":  msg.Text,
			"platform": msg.Platform,
			"error":    err.Error(),
		})
	}
}


func NewApp() *App {
	appDir, err := os.UserConfigDir()
	var appDataDir string
	if err == nil {
		appDataDir = filepath.Join(appDir, "chat-alert")
	}
	return NewAppWithDir(appDataDir)
}

func NewAppWithDir(appDataDir string) *App {
	cfg := config.Default()
	if appDataDir != "" {
		configPath := filepath.Join(appDataDir, "config.json")
		if loaded, err := config.Load(configPath); err == nil {
			cfg = loaded
		}
	}

	return &App{
		overlayVisible: false,
		cfg:            cfg,
		appDataDir:     appDataDir,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.initDirectories()
	a.setupLogger()

	if a.ctx != nil {
		func() {
			defer func() { recover() }()
			runtime.WindowCenter(ctx)
		}()
	}

	a.initIRC()
	a.initTTS()
	a.initQueue()
	a.initHotkey()

	if a.cfg.TwitchOAuthToken != "" && a.cfg.TwitchChannel != "" {
		go func() {
			if err := a.ircClient.client.Connect(); err != nil {
				log.Printf("failed to connect to Twitch: %v", err)
			} else {
				log.Printf("connected to Twitch channel: %s", a.cfg.TwitchChannel)
			}
		}()
	}

	// Start watching config file
	configPath := filepath.Join(a.appDataDir, "config.json")
	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		close(done)
	}()
	config.Watch(configPath, func(newCfg *config.Config) {
		log.Println("config file changed externally, reloading")
		a.ReloadConfig(newCfg)
	}, done)
}

func (a *App) initDirectories() {
	if a.appDataDir == "" {
		return
	}

	if err := os.MkdirAll(a.appDataDir, 0755); err != nil {
		log.Printf("failed to create app data dir: %v", err)
		return
	}

	a.logDir = filepath.Join(a.appDataDir, "logs")
	if err := os.MkdirAll(a.logDir, 0755); err != nil {
		log.Printf("failed to create log dir: %v", err)
		return
	}
}

func (a *App) setupLogger() {
	if a.logDir == "" {
		return
	}
	logFile := filepath.Join(a.logDir, "chat-alert.log")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("failed to open log file: %v", err)
		return
	}
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags)
}

func (a *App) ReloadConfig(cfg *config.Config) {
	a.applyConfig(cfg)
	runtime.EventsEmit(a.ctx, "config-reloaded", cfg)
}

func (a *App) applyConfig(cfg *config.Config) {
	a.mu.Lock()
	oldCfg := a.cfg
	a.cfg = cfg

	// Decide what needs to be restarted
	ircChanged := oldCfg.TwitchChannel != cfg.TwitchChannel || oldCfg.TwitchOAuthToken != cfg.TwitchOAuthToken
	ttsChanged := oldCfg.ThaiVoiceName != cfg.ThaiVoiceName || oldCfg.EnglishVoiceName != cfg.EnglishVoiceName || oldCfg.SpeechRateMultiplier != cfg.SpeechRateMultiplier || oldCfg.TTSEngine != cfg.TTSEngine || oldCfg.CloudTTSAPIKey != cfg.CloudTTSAPIKey || oldCfg.GeminiVoiceName != cfg.GeminiVoiceName || oldCfg.GeminiModel != cfg.GeminiModel
	queueDestructive := oldCfg.MaxQueueSize != cfg.MaxQueueSize

	if ircChanged && a.ircClient != nil && a.ircClient.client != nil {
		a.ircClient.client.Disconnect()
	}

	if (ttsChanged || queueDestructive) && a.msgQueue != nil {
		a.msgQueue.Stop()
	} else if a.msgQueue != nil {
		// Just update queue settings if not destructive
		a.msgQueue.UpdateConfig(queue.Config{
			MaxSize:       cfg.MaxQueueSize,
			AutoFadeDelay: cfg.AutoFadeDelay,
		})
	}
	a.mu.Unlock()

	if ircChanged {
		a.initIRC()
	}
	if ttsChanged || queueDestructive {
		if err := a.initTTS(); err != nil {
			log.Printf("failed to initialize TTS: %v", err)
		}
		a.initQueue()
	}

	if ircChanged && cfg.TwitchOAuthToken != "" && cfg.TwitchChannel != "" {
		go func() {
			if err := a.ircClient.client.Connect(); err != nil {
				log.Printf("failed to reconnect to Twitch: %v", err)
			} else {
				log.Printf("reconnected to Twitch channel: %s", cfg.TwitchChannel)
			}
		}()
	}

	if err := a.UpdateHotkeys(); err != nil {
		log.Printf("failed to update hotkeys: %v", err)
	}
}

func (a *App) initIRC() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ircClient = &ircClient{
		client: irc.NewClient(&ircConfigAdapter{a.cfg}),
	}
	a.ircClient.client.OnChatMessage(func(msg irc.ChatMessage) {
		log.Printf("received chat message from %s in %s: %s", msg.Username, msg.Channel, msg.Text)
		a.mu.RLock()
		q := a.msgQueue
		a.mu.RUnlock()
		if q != nil {
			q.Enqueue(msg)
		} else {
			log.Printf("warning: message queue not initialized")
		}
	})
	a.ircClient.client.OnConnected = func() {
		runtime.EventsEmit(a.ctx, "service-connected", map[string]string{
			"service": "twitch",
		})
	}
	a.ircClient.client.OnDisconnected = func(reason string) {
		runtime.EventsEmit(a.ctx, "service-disconnected", map[string]string{
			"service": "twitch",
			"reason":  reason,
		})
	}
	a.ircClient.client.OnReconnecting = func(attempt int, maxAttempts int) {
		runtime.EventsEmit(a.ctx, "service-reconnecting", map[string]interface{}{
			"service":      "twitch",
			"attempt":      attempt,
			"maxAttempts": maxAttempts,
		})
	}
}

func (a *App) initTTS() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	engine, err := tts.NewEngine(&tts.EngineConfig{
		SpeechRateMultiplier: a.cfg.SpeechRateMultiplier,
		CloudTTSEnabled:     a.cfg.TTSEngine == "cloud",
		CloudAPIKey:          a.cfg.CloudTTSAPIKey,
		ThaiVoiceName:        a.cfg.ThaiVoiceName,
		EnglishVoiceName:     a.cfg.EnglishVoiceName,
		GeminiVoiceName:      a.cfg.GeminiVoiceName,
		GeminiModel:          a.cfg.GeminiModel,
	})
	if err != nil {
		return err
	}
	a.ttsEngine = engine
	engine.SetErrorCallback(func(err error) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "tts-error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	})
	return nil
}

func (a *App) initQueue() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.msgQueue = queue.New(a.ttsEngine, queue.Config{
		MaxSize:       a.cfg.MaxQueueSize,
		AutoFadeDelay: a.cfg.AutoFadeDelay,
	})
	a.msgQueue.SetEmitter(a)
}

func (a *App) initHotkey() {
	a.hotkeyManager = hotkey.NewManager()

	if a.ctx != nil {
		a.hotkeyManager.Start(a.ctx)

		a.hotkeyManager.Register(a.ctx, a.cfg.ToggleOverlayHotkey, func() {
			a.ToggleOverlay()
		})

		a.hotkeyManager.Register(a.ctx, a.cfg.PinLastMessageHotkey, func() {
			a.PinLastMessage()
		})
	}
}

func (a *App) PinLastMessage() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.speakingActive && len(a.pinnedMessages) > 0 {
		if a.msgQueue != nil {
			a.msgQueue.Pause()
		}
		lastMsg := a.pinnedMessages[len(a.pinnedMessages)-1]
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "pin-message", map[string]string{
				"username": lastMsg.Username,
				"message":  lastMsg.Text,
				"platform": lastMsg.Platform,
			})
		}
	}
}

func (a *App) UnpinMessage() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.msgQueue != nil {
		a.msgQueue.Resume()
	}
}

func (a *App) PassMessage() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.msgQueue != nil {
		a.msgQueue.Resume()
		a.msgQueue.SkipCurrent()
	}
}

func (a *App) UpdateHotkeys() error {
	if a.hotkeyManager == nil || a.ctx == nil {
		return fmt.Errorf("hotkey manager not initialized")
	}

	a.hotkeyManager.UnregisterAll()

	if a.cfg.ToggleOverlayHotkey != "" {
		if err := a.hotkeyManager.Register(a.ctx, a.cfg.ToggleOverlayHotkey, func() {
			a.ToggleOverlay()
		}); err != nil {
			return fmt.Errorf("failed to register toggle overlay hotkey: %w", err)
		}
	}

	if a.cfg.PinLastMessageHotkey != "" {
		if err := a.hotkeyManager.Register(a.ctx, a.cfg.PinLastMessageHotkey, func() {
			a.PinLastMessage()
		}); err != nil {
			return fmt.Errorf("failed to register pin message hotkey: %w", err)
		}
	}

	return nil
}

func (a *App) RegisterHotkeys() error {
	if a.hotkeyManager == nil || a.ctx == nil {
		return fmt.Errorf("hotkey manager not initialized")
	}
	return nil
}

func (a *App) SavePosition(x, y int) error {
	a.mu.Lock()
	a.cfg.OverlayPosition.X = x
	a.cfg.OverlayPosition.Y = y
	cfg := a.cfg
	a.mu.Unlock()

	configPath := filepath.Join(a.appDataDir, "config.json")
	return config.Save(configPath, cfg)
}

func (a *App) SaveCurrentPosition() {
	if a.ctx == nil {
		return
	}

	a.mu.RLock()
	inSettings := a.settingsMode
	a.mu.RUnlock()

	if inSettings {
		return
	}

	x, y := runtime.WindowGetPosition(a.ctx)

	// Simple clamping to ensure it doesn't go fully off-screen (top-left)
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	a.SavePosition(x, y)
}

func (a *App) Connect(cfg *config.Config) error {
	if cfg.TwitchOAuthToken == "" {
		return fmt.Errorf("OAuth token required")
	}
	if cfg.TwitchChannel == "" {
		return fmt.Errorf("channel required")
	}

	ircCfg := &ircConfigAdapter{cfg}
	a.mu.Lock()
	a.ircClient.client = irc.NewClient(ircCfg)
	a.mu.Unlock()

	return a.ircClient.client.Connect()
}

func (a *App) Disconnect() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.ircClient != nil && a.ircClient.client != nil {
		a.ircClient.client.Disconnect()
	}
}

func (a *App) OnChatMessage(cb func(string, string)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.ircClient != nil && a.ircClient.client != nil {
		a.ircClient.client.OnChatMessage(func(msg irc.ChatMessage) {
			cb(msg.Username, msg.Text)
		})
	}
}

type ircConfigAdapter struct {
	cfg *config.Config
}

func (c *ircConfigAdapter) TwitchOAuthToken() string {
	token := c.cfg.TwitchOAuthToken
	if token != "" && !strings.HasPrefix(token, "oauth:") {
		return "oauth:" + token
	}
	return token
}

func (c *ircConfigAdapter) TwitchChannel() string {
	if c.cfg.TwitchChannel == "" {
		return ""
	}
	if c.cfg.TwitchChannel[0] != '#' {
		return "#" + c.cfg.TwitchChannel
	}
	return c.cfg.TwitchChannel
}

func (a *App) GetAppDataDir() string {
	return a.appDataDir
}

func (a *App) GetLogDir() string {
	return a.logDir
}

func (a *App) IsOverlay() bool {
	return a.overlayVisible
}

func (a *App) ShowOverlay() {
	a.mu.Lock()
	a.overlayVisible = true
	a.mu.Unlock()
	if a.ctx != nil {
		runtime.WindowShow(a.ctx)
	}
}

func (a *App) HideOverlay() {
	a.mu.Lock()
	if a.settingsMode {
		a.mu.Unlock()
		return
	}
	a.overlayVisible = false
	a.mu.Unlock()
	if a.ctx != nil {
		runtime.WindowHide(a.ctx)
	}
}

func (a *App) ToggleOverlay() {
	a.mu.Lock()
	visible := a.overlayVisible
	a.mu.Unlock()

	if visible {
		a.HideOverlay()
	} else {
		a.ShowOverlay()
	}
}

func (a *App) ToggleVisibility() {
	a.ToggleOverlay()
}

func (a *App) EnterSettingsMode() {
	a.mu.Lock()
	if a.settingsMode {
		a.mu.Unlock()
		return
	}
	a.settingsMode = true
	a.mu.Unlock()

	if a.ctx != nil {
		x, y := runtime.WindowGetPosition(a.ctx)
		a.mu.Lock()
		a.cfg.OverlayPosition.X = x
		a.cfg.OverlayPosition.Y = y
		a.mu.Unlock()

		runtime.WindowSetAlwaysOnTop(a.ctx, false)
		runtime.WindowSetSize(a.ctx, 500, 700)
		runtime.WindowCenter(a.ctx)
		runtime.EventsEmit(a.ctx, "settings-mode-active", nil)
	}
}

func (a *App) ResizeWindow(width, height int) {
	if a.ctx != nil {
		runtime.WindowSetSize(a.ctx, width, height)
	}
}

func (a *App) GetPinnedMessages() []irc.ChatMessage {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.pinnedMessages
}

func (a *App) ExitSettingsMode() {
	a.mu.Lock()
	if !a.settingsMode {
		a.mu.Unlock()
		return
	}
	a.settingsMode = false
	x := a.cfg.OverlayPosition.X
	y := a.cfg.OverlayPosition.Y
	a.mu.Unlock()

	if a.ctx != nil {
		runtime.WindowSetSize(a.ctx, 900, 350)
		runtime.WindowSetPosition(a.ctx, x, y)
		runtime.WindowSetAlwaysOnTop(a.ctx, true)
		runtime.EventsEmit(a.ctx, "settings-mode-inactive", nil)
	}
}


func (a *App) OpenSettings() {
	if a.ctx != nil {
		runtime.WindowShow(a.ctx)
		a.EnterSettingsMode()
		runtime.EventsEmit(a.ctx, "show-settings", nil)
	}
}

func (a *App) Quit() {
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}

func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) GetConfig() *config.Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg
}

func (a *App) CancelSettings() {
	a.ExitSettingsMode()
}

func (a *App) SaveConfig(cfg *config.Config) error {
	a.applyConfig(cfg)
	configPath := filepath.Join(a.appDataDir, "config.json")
	err := config.Save(configPath, cfg)
	if err == nil {
		a.ExitSettingsMode()
	}
	return err
}

type TTSInfo struct {
	Engine        string   `json:"engine"`
	ThaiVoices    []string `json:"thaiVoices"`
	EnglishVoices []string `json:"englishVoices"`
	GeminiVoices  []string `json:"geminiVoices"`
	Error         string   `json:"error"`
}

func (a *App) GetTTSInfo() TTSInfo {
	a.mu.RLock()
	engine := a.ttsEngine
	a.mu.RUnlock()

	info := TTSInfo{}

	if engine == nil {
		info.Error = "TTS engine not initialized"
		return info
	}

	thaiVoices, thaiErr := engine.ListVoices("th")
	if thaiErr != nil {
		info.Error = fmt.Sprintf("Failed to get Thai voices: %v", thaiErr)
	} else {
		info.ThaiVoices = thaiVoices
	}

	englishVoices, englishErr := engine.ListVoices("en")
	if englishErr != nil && info.Error == "" {
		info.Error = fmt.Sprintf("Failed to get English voices: %v", englishErr)
	} else if englishErr == nil {
		info.EnglishVoices = englishVoices
	}

	geminiVoices, geminiErr := engine.ListVoices("")
	if geminiErr == nil {
		info.GeminiVoices = geminiVoices
	}

	return info
}