# Setup and Run Guide for Chat Alert

This guide provides step-by-step instructions to set up and run the **Chat Alert** application on your Windows PC.

## 1. Prerequisites

Ensure you have the following installed:

- **Go (Golang):** Version 1.23.0 or higher.
  - [Download Go](https://go.dev/dl/)
- **Node.js & NPM:** Latest LTS version.
  - [Download Node.js](https://nodejs.org/)
- **C Compiler (gcc):** Required for Go's CGO (used for hotkeys and system tray).
  - Recommendation: Install [MSYS2](https://www.msys2.org/) and run `pacman -S mingw-w64-x86_64-toolchain`.
- **WebView2 Runtime:** Usually pre-installed on Windows 10/11.
  - [Download WebView2](https://developer.microsoft.com/en-us/microsoft-edge/webview2/)

## 2. Install Wails CLI

Wails is the framework used to build this app. Install the CLI tool by running:

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

_Note: Make sure your `go/bin` directory (usually `%USERPROFILE%\go\bin`) is in your system's PATH._

## 3. Configuration

The app stores its settings in your user profile.

1.  Navigate to `%APPDATA%\chat-alert\` (you can paste this into Windows Explorer).
2.  If it doesn't exist, create a folder named `chat-alert`.
3.  Create a file named `config.json` inside that folder with your Twitch details:

```json
{
  "twitchChannel": "#your_twitch_username",
  "twitchOAuthToken": "oauth:your_oauth_token",
  "autoFadeDelay": 5,
  "maxQueueSize": 100,
  "speechRateMultiplier": 1.0,
  "toggleOverlayHotkey": "Ctrl+Shift+T",
  "pinLastMessageHotkey": "Ctrl+Shift+P"
}
```

_You can get your OAuth token from [https://twitchapps.com/tmi/](https://twitchapps.com/tmi/)._

## 4. How to Run

### Option A: Development Mode (Best for Testing)

This mode allows you to see the app running and reloads automatically if you change the code.

1.  Open a terminal (PowerShell or CMD) in the project folder (`C:\Coding\chat-alert\chat-alert`).
2.  Run:
    ```powershell
    wails dev
    ```

### Option B: Build a Standalone Executable

This creates a single `.exe` file you can run anytime.

1.  Open a terminal in the project folder.
2.  Run:
    ```powershell
    wails build
    ```
3.  The executable will be generated in `build/bin/chat-alert.exe`.

## 5. Text-to-Speech Voices

### Installing Thai TTS Voice

The app supports Thai text-to-speech, but requires a Thai voice to be installed on your system. Follow these steps:

**Windows 10/11:**

1. Open **Settings** > **Time & Language** > **Speech**
2. Click **Add voice language** or **Manage voices**
3. Select **Thai (Thailand)** from the list and install
4. After installation, your Thai voice will appear in the app's settings dropdown

**Alternative using Speech Properties:**

1. Press `Win + X` and select **Control Panel**
2. Go to **Ease of Access** > **Speech Recognition** > **Text to Speech**
3. Click **Voice Settings** under the voice selection dropdown
4. Click **Add** to install additional voices
5. Select Thai from the available options

**Verifying Installation:**

To verify Thai voice is installed, open PowerShell and run:

```powershell
Add-Type -AssemblyName System.Speech
$synth = New-Object System.Speech.Synthesis.SpeechSynthesizer
$synth.GetInstalledVoices() | Where-Object { $_.VoiceInfo.Culture.Name -like 'th*' } | ForEach-Object { $_.VoiceInfo.Description }
```

## 6. Troubleshooting

### Installation Issues

#### "C compiler not found" or CGO errors

**Problem:** Build fails with `gcc: command not found` or CGO-related errors.

**Solution:**

- Ensure you've installed MSYS2 and the `mingw-w64-x86_64-toolchain`
- Verify `gcc` is in your PATH: Open PowerShell and run `gcc --version`
- If not found, add `C:\msys64\mingw64\bin` to your system PATH and restart your terminal

#### "go/bin is not in PATH"

**Problem:** `wails` command not found after installation.

**Solution:**

- Check your PATH: Run `echo $env:PATH` in PowerShell
- Add `$env:USERPROFILE\go\bin` to your system PATH environment variable
- Restart your terminal or IDE after updating PATH

#### WebView2 Runtime errors

**Problem:** App crashes with "WebView2 not found" or similar error.

**Solution:**

- [Download and install WebView2 Runtime](https://developer.microsoft.com/en-us/microsoft-edge/webview2/)
- Restart the application after installation

### Runtime Issues

#### No audio output from TTS

**Problem:** App runs but no sound is heard.

**Solution:**

- Verify your Windows audio output device is set correctly (Settings > Sound > Volume)
- Check the app's voice dropdown in settings — ensure a voice is selected
- Test with the system's default voice first
- Adjust the **Speech Rate Multiplier** in config if messages sound garbled

#### Overlay not appearing

**Problem:** App is running but the overlay doesn't show.

**Solution:**

- Check your hotkey binding: Press `Ctrl+Shift+T` (or your configured toggle hotkey)
- Verify the overlay visibility setting in the app's UI
- Try restarting the app
- Ensure your display driver supports frameless windows (rare issue)

#### Twitch connection fails

**Problem:** App starts but shows "Not connected" or chat isn't being read.

**Solution:**

- Verify your config.json has the correct Twitch username and OAuth token
- Re-generate your OAuth token at https://twitchapps.com/tmi/ (old tokens may expire)
- Ensure your Twitch username in `config.json` is lowercase and prefixed with `#` (e.g., `#mychannel`)
- Check your internet connection

#### Messages in queue not being read

**Problem:** Messages appear in the queue but don't get spoken.

**Solution:**

- Check that a voice is selected in the app's settings
- Verify **Max Queue Size** isn't set too low (try 50-100 for active chats)
- Try adjusting the **Auto Fade Delay** (default 5 seconds)
- Restart the app if stuck

### Thai Voice Issues

#### Thai voice won't appear in dropdown

**Problem:** You installed Thai language but it's not showing in settings.

**Solution:**

- Restart the app after installing the Thai voice
- Verify installation with the PowerShell script in Section 5
- Ensure Windows language pack for Thai is fully installed (may require restart)

## 7. FAQ

### Can I use this with other streaming platforms?

**Currently, Chat Alert only supports Twitch.** The architecture is modular, so adding other platforms (YouTube, Discord, etc.) is possible as future work.

### How do I change the hotkeys?

Edit your `config.json` file in `%APPDATA%\chat-alert\config.json`. Update the `toggleOverlayHotkey` and `pinLastMessageHotkey` fields. Restart the app for changes to take effect.

Example:

```json
"toggleOverlayHotkey": "Alt+O",
"pinLastMessageHotkey": "Alt+P"
```

### Can I run multiple instances?

**Not recommended.** The app stores state in `%APPDATA%\chat-alert\`, so running multiple instances may cause conflicts. Stick to one instance per Twitch channel.

### Where are logs stored?

Logs are stored in `%APPDATA%\chat-alert\logs\`. Check these if you're troubleshooting issues.

### How do I uninstall?

1. Close the app
2. Delete the executable (`build/bin/chat-alert.exe`)
3. **(Optional) Remove settings:** Delete `%APPDATA%\chat-alert\` (this will erase your config and logs)

### Can I build a standalone .exe for distribution?

**Yes.** Run `wails build` to create a release build in `build/bin/chat-alert.exe`. This is a standalone executable that doesn't require Go or Node.js to run.

### My config keeps resetting

**This shouldn't happen.** If your config is being reset:

1. Ensure the file is saved as `config.json` (not `config.json.txt`)
2. Verify the file is valid JSON (use an online JSON validator)
3. Check file permissions — the app needs write access to `%APPDATA%\chat-alert\`
4. Check the logs folder for error messages

### Can I customize the overlay appearance?

Currently, the overlay appearance (colors, fonts, layout) is hardcoded. Customization via config is on the roadmap.

If no Thai voice is installed:

- `GetVoices('th')` in the app will return an empty list
- Thai text will not be spoken until a Thai voice is added

### Available Voices

The app will automatically detect and list all installed voices. In the app settings, you'll see:

- **Thai Voice**: Dropdown populated with all installed Thai voices
- **English Voice**: Dropdown populated with all installed English voices

---

## 6. Troubleshooting

- **Missing GCC:** If you get a "gcc not found" error, ensure your C compiler's `bin` folder is in your PATH.
- **Environment Check:** Run `wails doctor` to see if your system meets all requirements.
- **Hotkeys:** If hotkeys don't work, ensure no other application is using the same key combination.
- **Thai TTS not working:** Verify a Thai voice is installed in Windows Speech settings (see Section 5 above).

---

_Created by Gemini CLI_
