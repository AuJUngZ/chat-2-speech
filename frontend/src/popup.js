import "./popup.css";
import {
  GetConfig,
  SaveConfig,
  GetTTSInfo,
  EnterSettingsMode,
  CancelSettings,
  GetPinnedMessages,
  PinLastMessage,
  UnpinMessage,
  PassMessage,
} from "../wailsjs/go/main/App";

let settingsVisible = false;
let activeMessage = null;
let queuedMessages = [];
let historyStack = [];
let isPinned = false;
const MAX_HISTORY = 20;

export function showToast(message, type = "info") {
  const existingToast = document.querySelector(".toast");
  if (existingToast) existingToast.remove();

  const toast = document.createElement("div");
  toast.className = `toast toast-${type}`;
  toast.textContent = message;

  if (type === "error") {
    const retryBtn = document.createElement("button");
    retryBtn.className = "toast-retry-btn";
    retryBtn.textContent = "Retry";
    retryBtn.addEventListener("click", () => {
      dismissToast(toast);
    });
    toast.appendChild(retryBtn);

    const dismissBtn = document.createElement("button");
    dismissBtn.className = "toast-dismiss-btn";
    dismissBtn.textContent = "✕";
    dismissBtn.addEventListener("click", () => dismissToast(toast));
    toast.appendChild(dismissBtn);
  }

  toast.addEventListener("mouseenter", () => {
    toast.style.animationPlayState = "paused";
  });
  toast.addEventListener("mouseleave", () => {
    toast.style.animationPlayState = "running";
  });

  document.body.appendChild(toast);

  if (type !== "error") {
    setTimeout(() => dismissToast(toast), 3000);
  }
}

function dismissToast(toast) {
  if (!toast.parentNode) return;
  toast.classList.add("toast-out");
  setTimeout(() => {
    if (toast.parentNode) toast.remove();
  }, 1000);
}

const getAppElement = () => document.getElementById("app") || document.body;

export function resetModuleState() {
  settingsVisible = false;
  activeMessage = null;
  queuedMessages = [];
  historyStack = [];
  isPinned = false;
  const app = getAppElement();
  if (app) {
    app.innerHTML = "";
  }
  initLayout();
}

function getPlatformIcon(platform) {
  switch (platform?.toLowerCase()) {
    case "twitch":
      return "ri-twitch-fill";
    case "youtube":
      return "ri-youtube-fill";
    case "tiktok":
      return "ri-tiktok-fill";
    default:
      return "ri-chat-3-line";
  }
}

function initLayout() {
  const container = getAppElement();
  if (!container || document.querySelector(".sidebar")) return;

  const sidebar = document.createElement("div");
  sidebar.className = "sidebar";
  sidebar.innerHTML = `
    <div class="history-section">
      <div class="section-header">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
        <span class="section-title">Spoken History</span>
      </div>
      <div class="history-list"></div>
    </div>
  `;
  container.appendChild(sidebar);

  const chatContainer = document.createElement("div");
  chatContainer.className = "chat-container";
  chatContainer.innerHTML = '<div class="stack-container"></div>';
  container.appendChild(chatContainer);

  renderStack();
  renderHistory();

  if (GetPinnedMessages) {
    GetPinnedMessages()
      .then((msgs) => {
        if (msgs && msgs.length > 0) {
          historyStack = msgs.map((m) => ({
            username: m.Username,
            message: m.Text,
            platform: m.Platform,
          }));
          renderHistory();
        }
      })
      .catch(() => {});
  }
}

function createCard({ username, message, platform, isActive }, index) {
  const card = document.createElement("div");
  card.className = `chat-popup stack-${index}`;
  if (isActive && isPinned) card.classList.add("pinned");

  card.innerHTML = `
    <div class="popup-actions">
      <div class="popup-badge">${isActive ? "Speaking Now" : "Upcoming"}</div>
      ${
        isActive
          ? `
      <div class="popup-btn-group">
        <button class="popup-btn popup-btn-pin ${isPinned ? "active" : ""}">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m21.44 11.05-9.19 9.19a6 6 0 0 1-8.49-8.49l9.19-9.19a4 4 0 0 1 5.66 5.66l-9.2 9.19a2 2 0 0 1-2.83-2.83l8.49-8.48"/></svg>
          <span>${isPinned ? "Unpin" : "Pin"}</span>
        </button>
        <button class="popup-btn popup-btn-pass">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
          <span>Pass</span>
        </button>
      </div>
      `
          : ""
      }
    </div>
    <div class="popup-content">
      <div class="popup-username-row">
        <div class="popup-username">${username}</div>
        <div class="platform-badge platform-${platform || "unknown"}">
          <i class="${getPlatformIcon(platform)}"></i>
          <span>${platform || "chat"}</span>
        </div>
      </div>
      <div class="popup-message">${message}</div>
    </div>
  `;

  if (isActive) {
    card.querySelector(".popup-btn-pin").addEventListener("click", () => {
      if (PinLastMessage) PinLastMessage();
    });
    card.querySelector(".popup-btn-pass").addEventListener("click", () => {
      isPinned = false;
      card.classList.add("fade-out");
      hidePopup();
      if (PassMessage) PassMessage();
    });
  }

  return card;
}

export async function checkEmptyState() {
  const emptyState = document.querySelector('[data-check-config="true"]');
  if (!emptyState) return;

  const config = await GetConfig();
  const isConfigured = config.twitchOAuthToken && config.twitchChannel;

  if (!isConfigured) {
    emptyState.innerHTML = `
      <div class="empty-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/></svg>
      </div>
      <div class="empty-text">No chat connections configured.</div>
      <div class="empty-subtext">Configure your chat connection to get started.</div>
    `;
  } else {
    emptyState.innerHTML = `
      <div class="empty-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/></svg>
      </div>
      <div class="empty-text">Waiting for messages...</div>
      <div class="empty-subtext">New chat messages will appear here</div>
    `;
  }
}

export function renderStack() {
  const container = document.querySelector(".stack-container");
  if (!container) return;

  const stack = [];
  if (activeMessage) {
    stack.push({ ...activeMessage, isActive: true });
  }

  for (let i = 0; i < queuedMessages.length && stack.length < 5; i++) {
    const qMsg = {
      username: queuedMessages[i].username,
      message: queuedMessages[i].message,
      platform: queuedMessages[i].platform,
      isActive: false,
    };

    if (
      activeMessage &&
      qMsg.username === activeMessage.username &&
      qMsg.message === activeMessage.message
    ) {
      continue;
    }
    stack.push(qMsg);
  }

  container.innerHTML = "";

  if (stack.length === 0) {
    container.classList.add("is-empty");
    const emptyState = document.createElement("div");
    emptyState.className = "empty-stack-state";
    emptyState.setAttribute("data-check-config", "true");
    emptyState.innerHTML = `
      <div class="empty-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/></svg>
      </div>
      <div class="empty-text">Waiting for messages...</div>
      <div class="empty-subtext">New chat messages will appear here</div>
    `;
    container.appendChild(emptyState);
    checkEmptyState();
    return;
  }

  container.classList.remove("is-empty");
  const GAP = 25; // Vertical gap between stacked items

  stack.forEach((item, index) => {
    const card = createCard(item, index);

    // index 0 (Active) is anchored at the "top" of our stack visualization.
    // index 1+ (Upcoming) stack DOWNWARDS (+Y) and BEHIND the active one.

    // Position active message at y=0, others offset downwards (+Y)
    const yOffset = index * GAP;

    const scale = 1 - index * 0.04;
    const opacity = 1 - index * 0.2;
    const brightness = 1 - index * 0.1;
    const zIndex = 50 - index;

    card.style.setProperty("--y-offset", `${yOffset}px`);
    card.style.setProperty("--scale", scale);
    card.style.setProperty("--opacity", opacity);
    card.style.setProperty("--brightness", brightness);
    card.style.setProperty("--z-index", zIndex);

    container.appendChild(card);
  });
}

export function showPopup({ username, message, platform }) {
  initLayout();
  addToHistory(username, message, platform);
  activeMessage = { username, message, platform };
  renderStack();
}

export function hidePopup() {
  if (activeMessage && !isPinned) {
    activeMessage = null;
    renderStack();
  }
}

export function updateQueue(messages) {
  queuedMessages = messages || [];
  renderStack();
}

export function addToHistory(username, message, platform) {
  if (historyStack.length > 0) {
    const last = historyStack[historyStack.length - 1];
    if (last.username === username && last.message === message) return;
  }

  historyStack.push({ username, message, platform });
  if (historyStack.length > MAX_HISTORY) {
    historyStack.shift();
  }
  renderHistory();
}

export function renderHistory() {
  const container = document.querySelector(".history-list");
  if (!container) return;

  if (historyStack.length === 0) {
    container.innerHTML = `
      <div class="empty-history-state">
        <span>No history yet</span>
      </div>
    `;
    return;
  }

  container.innerHTML = historyStack
    .slice()
    .reverse()
    .map(
      (m) => `
    <div class="history-item">
      <div class="history-username-row">
        <span class="history-username">${m.username}</span>
        <div class="platform-badge platform-${m.platform} is-mini">
          <i class="${getPlatformIcon(m.platform)}"></i>
          <span>${m.platform}</span>
        </div>
      </div>
      <span class="history-message">${m.message}</span>
    </div>
  `,
    )
    .join("");
}

export function setPinned(pinned) {
  isPinned = pinned;
  renderStack();
}

export function pinCurrent() {
  if (!activeMessage) return;
  isPinned = !isPinned;
  if (!isPinned && UnpinMessage) UnpinMessage();
  renderStack();
}

export function unpinCurrent() {
  isPinned = false;
  if (UnpinMessage) UnpinMessage();
  renderStack();
}

export function unpinMessage(username, message) {
  if (
    activeMessage &&
    activeMessage.username === username &&
    activeMessage.message === message
  ) {
    unpinCurrent();
  }
}

export async function toggleSettings() {
  const existing = document.querySelector(".settings-panel");
  if (existing) {
    existing.remove();
    settingsVisible = false;
    await CancelSettings();
    return;
  }

  settingsVisible = true;
  await EnterSettingsMode();

  let config, ttsInfo;
  try {
    config = await GetConfig();
    ttsInfo = await GetTTSInfo();
  } catch (e) {
    console.error("Failed to load settings data:", e);
    showToast("Failed to load settings: " + e.message, "error");
    settingsVisible = false;
    await CancelSettings();
    return;
  }

  const thaiVoices = ttsInfo.thaiVoices || [];
  const englishVoices = ttsInfo.englishVoices || [];
  const geminiVoices = ttsInfo.geminiVoices || [];
  const ttsError = ttsInfo.error || "";

  const panel = document.createElement("div");
  panel.className = "settings-panel";
  panel.innerHTML = `
    <div class="settings-container">
      <div class="settings-header">
        <h3>Settings</h3>
      </div>

      <div class="settings-tabs">
        <button class="tab-btn active" data-tab="config">Config</button>
        <button class="tab-btn" data-tab="hotkeys">Hotkeys</button>
      </div>
      
      <div id="tab-config" class="tab-content active">
        <div class="settings-section">
          <div class="section-label">Chat Connections</div>
          <div class="connection-grid">
            <div class="connection-card active">
              <div class="provider-info">
                <div class="provider-icon">
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M11.571 4.714h1.715v5.143H11.57zm4.715 0H18v5.143h-1.714zM6 0L1.714 4.286v15.428h5.143V24l4.286-4.286h3.428L22.286 12V0zm14.571 11.143l-3.428 3.428h-3.429l-3 3v-3H6.857V1.714h13.714Z"/></svg>
                </div>
                <span class="provider-name">Twitch</span>
              </div>
              <div class="settings-field">
                <label>OAuth Token</label>
                <input type="password" id="twitch-token" value="${config.twitchOAuthToken || ""}" placeholder="oauth:xxxxxx" />
              </div>
              <div class="settings-field">
                <label>Channel Name</label>
                <input type="text" id="twitch-channel" value="${config.twitchChannel || ""}" placeholder="my_channel" />
              </div>
            </div>
            
            <div class="connection-card" style="opacity: 0.5; cursor: not-allowed;">
              <div class="provider-info">
                <div class="provider-icon">
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M23.498 6.186a3.016 3.016 0 0 0-2.122-2.136C19.505 3.545 12 3.545 12 3.545s-7.505 0-9.377.505A3.017 3.017 0 0 0 .502 6.186C0 8.07 0 12 0 12s0 3.93.502 5.814a3.016 3.016 0 0 0 2.122 2.136c1.871.505 9.376.505 9.376.505s7.505 0 9.377-.505a3.015 3.015 0 0 0 2.122-2.136C24 15.93 24 12 24 12s0-3.93-.502-5.814zM9.545 15.568V8.432L15.818 12l-6.273 3.568z"/></svg>
                </div>
                <span class="provider-name">YouTube (Coming Soon)</span>
              </div>
            </div>
          </div>
        </div>

        <div class="settings-section">
          <div class="section-label">Voice & Audio</div>
          ${ttsError ? `<div class="tts-error">${ttsError}</div>` : ""}
          <div class="settings-field">
            <label>TTS Engine</label>
            <select id="tts-engine">
              <option value="local" ${config.ttsEngine === "local" ? "selected" : ""}>Local (Windows SAPI)</option>
              <option value="cloud" ${config.ttsEngine === "cloud" ? "selected" : ""}>Cloud (Gemini TTS)</option>
            </select>
          </div>
          <div class="settings-field" id="cloud-api-key-field" style="display: ${config.ttsEngine === "cloud" ? "block" : "none"}">
            <label>Cloud TTS API Key <span style="color: red;">*</span></label>
            <input type="password" id="cloud-api-key" value="${config.cloudTTSAPIKey || ""}" placeholder="Gemini API Key" />
          </div>
          <div class="grid-2" id="local-voice-fields">
            <div class="settings-field">
              <label>Thai Voice</label>
              <select id="thai-voice">
                ${thaiVoices.map((v) => `<option value="${v}" ${v === config.thaiVoiceName ? "selected" : ""}>${v}</option>`).join("")}
              </select>
            </div>
            <div class="settings-field">
              <label>English Voice</label>
              <select id="english-voice">
                ${englishVoices.map((v) => `<option value="${v}" ${v === config.englishVoiceName ? "selected" : ""}>${v}</option>`).join("")}
              </select>
            </div>
          </div>
          <div class="settings-field" id="cloud-voice-field" style="display: ${config.ttsEngine === "cloud" ? "block" : "none"}">
            <label>Model</label>
            <select id="gemini-model">
              <option value="gemini-2.5-flash-preview-tts" ${config.geminiModel === "gemini-2.5-flash-preview-tts" ? "selected" : ""}>gemini-2.5-flash-preview-tts</option>
              <option value="gemini-3.1-flash-tts-preview" ${!config.geminiModel || config.geminiModel === "gemini-3.1-flash-tts-preview" ? "selected" : ""}>gemini-3.1-flash-tts-preview (default)</option>
            </select>
          </div>
          <div class="settings-field" id="cloud-voice-field" style="display: ${config.ttsEngine === "cloud" ? "block" : "none"}">
            <label>Voice Type</label>
            <select id="gemini-voice">
              ${geminiVoices.map((v) => `<option value="${v}" ${v === config.geminiVoiceName ? "selected" : ""}>${v}</option>`).join("")}
            </select>
          </div>
          
          <div class="settings-field">
            <label id="speech-rate-label">Speech Rate (${config.speechRateMultiplier}x)</label>
            <input type="range" id="speech-rate" min="0.5" max="2.0" step="0.1" value="${config.speechRateMultiplier}" />
          </div>
        </div>

        <div class="settings-section">
          <div class="section-label">System Performance</div>
          <div class="grid-2">
            <div class="settings-field">
              <label>Auto-Fade Delay (sec)</label>
              <input type="number" id="auto-fade" value="${config.autoFadeDelay}" />
            </div>
            <div class="settings-field">
              <label>Max Queue Size</label>
              <input type="number" id="max-queue" value="${config.maxQueueSize}" />
            </div>
          </div>
          </div>
        </div>
      </div>

      <div id="tab-hotkeys" class="tab-content">
        <div class="settings-section">
          <div class="section-label">Registered Hotkeys</div>
          <div class="hotkey-list">
            <div class="hotkey-item">
              <span class="hotkey-label">Toggle Overlay</span>
              <span class="hotkey-value">${config.toggleOverlayHotkey || "Not Set"}</span>
            </div>
            <div class="hotkey-item">
              <span class="hotkey-label">Pin/Unpin Last Message</span>
              <span class="hotkey-value">${config.pinLastMessageHotkey || "Not Set"}</span>
            </div>
          </div>
          <p style="font-size: 12px; color: var(--text-muted); margin: 0; padding-top: 8px;">Hotkeys are read-only here. To change them, please edit the config.json file directly.</p>
        </div>
      </div>

      <div class="settings-actions">
        <button id="save-settings" class="btn-save">Save Changes</button>
        <button id="cancel-settings" class="btn-cancel">Cancel</button>
      </div>
    </div>
  `;

  const tabs = panel.querySelectorAll(".tab-btn");
  const contents = panel.querySelectorAll(".tab-content");

  tabs.forEach((tab) => {
    tab.addEventListener("click", () => {
      const target = tab.dataset.tab;
      tabs.forEach((t) => t.classList.remove("active"));
      contents.forEach((c) => c.classList.remove("active"));
      tab.classList.add("active");
      panel.querySelector(`#tab-${target}`).classList.add("active");
    });
  });

  const rateInput = panel.querySelector("#speech-rate");
  const rateLabel = panel.querySelector("#speech-rate-label");
  rateInput.addEventListener("input", (e) => {
    rateLabel.textContent = `Speech Rate (${e.target.value}x)`;
  });

  const ttsEngineSelect = panel.querySelector("#tts-engine");
  const cloudApiKeyField = panel.querySelector("#cloud-api-key-field");
  const localVoiceFields = panel.querySelector("#local-voice-fields");
  const cloudVoiceField = panel.querySelector("#cloud-voice-field");
  const thaiVoiceSelect = panel.querySelector("#thai-voice");
  const englishVoiceSelect = panel.querySelector("#english-voice");
  const geminiVoiceSelect = panel.querySelector("#gemini-voice");

  const updateVoiceOptions = async (engineType) => {
    const info = await GetTTSInfo();
    const thaiVoices = info.thaiVoices || [];
    const englishVoices = info.englishVoices || [];
    const geminiVoices = info.geminiVoices || [];

    const isCloud = engineType === "cloud";

    thaiVoiceSelect.innerHTML = thaiVoices.map((v) => `<option value="${v}">${v}</option>`).join("");
    englishVoiceSelect.innerHTML = englishVoices.map((v) => `<option value="${v}">${v}</option>`).join("");
    geminiVoiceSelect.innerHTML = geminiVoices.map((v) => `<option value="${v}">${v}</option>`).join("");

    if (isCloud) {
      thaiVoiceSelect.value = config.thaiVoiceName || "";
      englishVoiceSelect.value = config.englishVoiceName || "";
      geminiVoiceSelect.value = config.geminiVoiceName || geminiVoices[0] || "";
    } else {
      thaiVoiceSelect.value = config.thaiVoiceName || thaiVoices[0] || "";
      englishVoiceSelect.value = config.englishVoiceName || englishVoices[0] || "";
      geminiVoiceSelect.value = config.geminiVoiceName || "";
    }
  };

  ttsEngineSelect.addEventListener("change", async (e) => {
    applyVoiceFieldVisibility(e.target.value);
    await updateVoiceOptions(e.target.value);
  });

  await updateVoiceOptions(config.ttsEngine);

  const applyVoiceFieldVisibility = (engineType) => {
    const isCloud = engineType === "cloud";
    cloudApiKeyField.style.display = isCloud ? "block" : "none";
    localVoiceFields.style.display = isCloud ? "none" : "grid";
    cloudVoiceField.style.display = isCloud ? "block" : "none";
  };
  applyVoiceFieldVisibility(config.ttsEngine);

  panel.querySelector("#save-settings").addEventListener("click", async () => {
    const ttsEngine = panel.querySelector("#tts-engine").value;
    const cloudApiKey = panel.querySelector("#cloud-api-key").value;
    if (ttsEngine === "cloud" && !cloudApiKey.trim()) {
      showToast("Please provide Gemini API Key for Cloud TTS", "error");
      panel.querySelector("#cloud-api-key").focus();
      return;
    }

    const newConfig = {
      ...config,
      twitchOAuthToken: panel.querySelector("#twitch-token").value,
      twitchChannel: panel.querySelector("#twitch-channel").value,
      ttsEngine: ttsEngine,
      thaiVoiceName: panel.querySelector("#thai-voice").value,
      englishVoiceName: panel.querySelector("#english-voice").value,
      geminiVoiceName: panel.querySelector("#gemini-voice").value,
      geminiModel: panel.querySelector("#gemini-model").value,
      speechRateMultiplier: parseFloat(
        panel.querySelector("#speech-rate").value,
      ),
      autoFadeDelay: parseInt(panel.querySelector("#auto-fade").value, 10),
      maxQueueSize: parseInt(panel.querySelector("#max-queue").value, 10),
      cloudTTSAPIKey: cloudApiKey,
    };
    await SaveConfig(newConfig);
    panel.remove();
    settingsVisible = false;
    await CancelSettings();
    checkEmptyState();
  });

  panel
    .querySelector("#cancel-settings")
    .addEventListener("click", async () => {
      panel.remove();
      settingsVisible = false;
      await CancelSettings();
    });

  document.body.appendChild(panel);
}

export function removePopup(card) {
  card.classList.add("fade-out");
  setTimeout(() => {
    if (card.parentNode) card.remove();
  }, 300);
}

initLayout();
