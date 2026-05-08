# Chat Alert

**Read Twitch chat messages aloud with a sleek overlay and customizable voices.**

Chat Alert connects to your Twitch channel, reads incoming chat messages using Windows text-to-speech, and displays them in a frameless, always-on-top overlay. Perfect for streamers who want to engage with chat without constantly reading the screen.

## Features

- 🎙️ **Real-time TTS** — Reads chat messages aloud using Windows voices (supports Thai and many other languages)
- 👁️ **Overlay UI** — Sleek, frameless overlay that shows the current message and upcoming queue
- ⌨️ **Hotkeys** — Quickly toggle the overlay and pin important messages
- 📋 **Message Queue** — Manages message playback with customizable queue size
- ⚙️ **Easy Setup** — Simple configuration with your Twitch credentials
- 🎨 **Customizable** — Adjust speech rate, fade delay, and hotkey bindings

## Quick Start

1. **Install prerequisites** — Go 1.23+, Node.js LTS, C compiler, WebView2
2. **Install Wails** — `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
3. **Configure** — Create `%APPDATA%\chat-alert\config.json` with your Twitch details (see [SETUP.md](SETUP.md))
4. **Run** — `wails dev` (development) or `wails build` (production)

## Detailed Setup

For step-by-step installation, configuration, and troubleshooting, see [SETUP.md](SETUP.md).

## Controls

- **Ctrl+Shift+T** — Toggle overlay visibility
- **Ctrl+Shift+P** — Pin/unpin the current message

## License

MIT
