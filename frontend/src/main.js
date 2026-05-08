import './style.css';
import './app.css';

import { SaveCurrentPosition, ToggleOverlay, IsOverlay, ShowOverlay, HideOverlay } from '../wailsjs/go/main/App';
import { WindowShow, WindowHide, EventsOn } from '../wailsjs/runtime';
import { showPopup, hidePopup, toggleSettings, pinCurrent, showToast } from './popup.js';

export async function toggleOverlay() {
  const visible = await IsOverlay();
  if (visible) {
    await HideOverlay();
    await WindowHide();
  } else {
    await ShowOverlay();
    await WindowShow();
  }
}
window.toggleOverlay = toggleOverlay;

// Settings Button
const settingsBtn = document.createElement('button');
settingsBtn.id = 'settings-btn';
settingsBtn.innerHTML = `
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/>
    <circle cx="12" cy="12" r="3"/>
  </svg>
`;
settingsBtn.title = 'Settings';
settingsBtn.addEventListener('click', () => toggleSettings());
document.body.appendChild(settingsBtn);

function debounce(func, timeout = 500) {
  let timer;
  return (...args) => {
    clearTimeout(timer);
    timer = setTimeout(() => { func.apply(this, args); }, timeout);
  };
}

const debouncedSavePosition = debounce(() => {
  if (SaveCurrentPosition) SaveCurrentPosition();
});

window.addEventListener('mouseup', debouncedSavePosition);

export function initKeyboardShortcuts() {
  document.addEventListener('keydown', async (e) => {
    if (e.key === 'Escape') {
      const settingsPanel = document.querySelector('.settings-panel');
      if (settingsPanel) {
        toggleSettings();
      } else {
        await HideOverlay();
        await WindowHide();
      }
    }
  });
}


export function initTTSLifecycleHandlers() {
  EventsOn('tts-started', (data) => {
    showPopup({ username: data.username, message: data.message, platform: data.platform });
  });

  EventsOn('tts-fade-start', () => {
    hidePopup();
  });

  EventsOn('queue-updated', (messages) => {
    import('./popup.js').then(m => m.updateQueue(messages));
  });

  EventsOn('pin-message', () => {
    pinCurrent();
  });

  EventsOn('show-settings', () => {
    toggleSettings();
  });

  EventsOn('settings-mode-active', () => {
    document.body.classList.add('settings-mode');
    settingsBtn.style.display = 'none';
    const app = document.getElementById('app');
    if (app) app.style.visibility = 'hidden';
  });

  EventsOn('settings-mode-inactive', () => {
    document.body.classList.remove('settings-mode');
    settingsBtn.style.display = 'flex';
    const app = document.getElementById('app');
    if (app) app.style.visibility = 'visible';
  });
}

export function initServiceEventHandlers() {
  EventsOn('service-connected', (data) => {
    showToast(`Connected to ${data.service}`, 'success');
  });

  EventsOn('service-disconnected', (data) => {
    showToast(`Disconnected from ${data.service}`, 'error');
  });

  EventsOn('service-reconnecting', (data) => {
    showToast(`Reconnecting to ${data.service} (attempt ${data.attempt}/${data.maxAttempts})`, 'info');
  });
}

initKeyboardShortcuts();
initTTSLifecycleHandlers();
initServiceEventHandlers();

console.log('chat-alert modern overlay loaded');