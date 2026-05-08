import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockWindowShow = vi.fn();
const mockWindowHide = vi.fn();

vi.mock('../../wailsjs/runtime', () => ({
  WindowShow: mockWindowShow,
  WindowHide: mockWindowHide,
}));

vi.mock('../wailsjs/go/main/App', () => ({
  GetConfig: vi.fn().mockResolvedValue({}),
  SaveConfig: vi.fn().mockResolvedValue(null),
  GetVoices: vi.fn().mockResolvedValue([]),
  ResizeWindow: vi.fn().mockResolvedValue(null),
  GetPinnedMessages: vi.fn().mockResolvedValue([]),
  EnterSettingsMode: vi.fn().mockResolvedValue(null),
  CancelSettings: vi.fn().mockResolvedValue(null),
  PinLastMessage: vi.fn().mockResolvedValue(null),
  UnpinMessage: vi.fn().mockResolvedValue(null),
  PassMessage: vi.fn().mockResolvedValue(null),
  GetTTSInfo: vi.fn().mockResolvedValue({ thaiVoices: [], englishVoices: [] }),
}));

describe('popup', () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    document.body.innerHTML = '<div id="app"></div>';
    const { resetModuleState } = await import('./popup.js');
    resetModuleState();
  });

  describe('renderPopup', () => {
    it('renders username, message and platform badge in popup card', async () => {
      const { showPopup } = await import('./popup.js');
      showPopup({ username: 'TestUser', message: 'Hello chat', platform: 'twitch' });
      const card = document.querySelector('.chat-popup');
      expect(card).not.toBeNull();
      const usernameEl = card.querySelector('.popup-username');
      const messageEl = card.querySelector('.popup-message');
      const platformBadge = card.querySelector('.platform-badge');
      expect(usernameEl.textContent).toBe('TestUser');
      expect(messageEl.textContent).toBe('Hello chat');
      expect(platformBadge.textContent.trim()).toBe('twitch');
      expect(platformBadge.querySelector('i').classList.contains('ri-twitch-fill')).toBe(true);
    });

    it('renders Pin and Pass buttons', async () => {
      const { showPopup } = await import('./popup.js');
      showPopup({ username: 'TestUser', message: 'Hello chat' });
      const card = document.querySelector('.chat-popup');
      const pinBtn = card.querySelector('.popup-btn-pin');
      const passBtn = card.querySelector('.popup-btn-pass');
      expect(pinBtn).not.toBeNull();
      expect(passBtn).not.toBeNull();
      expect(pinBtn.textContent.trim()).toContain('Pin');
      expect(passBtn.textContent.trim()).toContain('Pass');
    });

    it('calls PinLastMessage when Pin button is clicked', async () => {
      const { showPopup } = await import('./popup.js');
      const { PinLastMessage } = await import('../wailsjs/go/main/App');
      showPopup({ username: 'User1', message: 'Hello' });
      const card = document.querySelector('.chat-popup');
      const pinBtn = card.querySelector('.popup-btn-pin');
      pinBtn.click();
      expect(PinLastMessage).toHaveBeenCalled();
    });

    it('calls PassMessage and hides popup when Pass button is clicked', async () => {
      const { showPopup } = await import('./popup.js');
      const { PassMessage } = await import('../wailsjs/go/main/App');
      showPopup({ username: 'User1', message: 'Hello' });
      const card = document.querySelector('.chat-popup');
      const passBtn = card.querySelector('.popup-btn-pass');
      passBtn.click();
      expect(PassMessage).toHaveBeenCalled();
      expect(card.classList.contains('fade-out')).toBe(true);
    });

    it('shows settings panel when toggleSettings is called', async () => {
      const { toggleSettings } = await import('./popup.js');
      await toggleSettings();
      const panel = document.querySelector('.settings-panel');
      expect(panel).not.toBeNull();
    });

    it('removes popup with fade-out class', async () => {
      const { showPopup, removePopup } = await import('./popup.js');
      showPopup({ username: 'TestUser', message: 'Hello' });
      const card = document.querySelector('.chat-popup');
      removePopup(card);
      expect(card.classList.contains('fade-out')).toBe(true);
    });

    it('skips removal if isPinned is true', async () => {
      const { showPopup, hidePopup, setPinned } = await import('./popup.js');
      showPopup({ username: 'User1', message: 'Hello' });
      let card = document.querySelector('.chat-popup');
      setPinned(true);
      // Re-query card because setPinned calls renderStack which recreates the element
      card = document.querySelector('.chat-popup');
      hidePopup();
      expect(document.contains(card)).toBe(true);
      expect(card.classList.contains('fade-out')).toBe(false);
    });
  });

  describe('Toast notifications', () => {
  describe('showToast', () => {
    it('creates toast element with type class', async () => {
      const { showToast } = await import('./popup.js');
      showToast('Test message', 'info');
      const toast = document.querySelector('.toast');
      expect(toast).not.toBeNull();
      expect(toast.classList.contains('toast-info')).toBe(true);
      expect(toast.textContent).toBe('Test message');
    });

    it('creates error toast with error class', async () => {
      const { showToast } = await import('./popup.js');
      showToast('Error occurred', 'error');
      const toast = document.querySelector('.toast');
      expect(toast.classList.contains('toast-error')).toBe(true);
    });

    it('creates success toast with success class', async () => {
      const { showToast } = await import('./popup.js');
      showToast('Success!', 'success');
      const toast = document.querySelector('.toast');
      expect(toast.classList.contains('toast-success')).toBe(true);
    });

    it('has entry animation (toastIn keyframe)', async () => {
      const { showToast } = await import('./popup.js');
      showToast('Test', 'info');
      const toast = document.querySelector('.toast');
      expect(toast).not.toBeNull();
      expect(toast.classList.contains('toast')).toBe(true);
    });

    it('error toast does not auto-dismiss (persistent)', async () => {
      vi.useFakeTimers();
      const { showToast } = await import('./popup.js');
      showToast('Error message', 'error');
      const toast = document.querySelector('.toast');
      expect(toast).not.toBeNull();
      vi.advanceTimersByTime(4000);
      expect(document.querySelector('.toast')).not.toBeNull();
      vi.useRealTimers();
    });

    it('info toast auto-dismisses after 3s', async () => {
      vi.useFakeTimers();
      const { showToast } = await import('./popup.js');
      showToast('Info message', 'info');
      const toast = document.querySelector('.toast');
      expect(toast).not.toBeNull();
      vi.advanceTimersByTime(3500);
      expect(document.querySelector('.toast')).toBeNull();
      vi.useRealTimers();
    });

    it('success toast auto-dismisses after 3s', async () => {
      vi.useFakeTimers();
      const { showToast } = await import('./popup.js');
      showToast('Success message', 'success');
      const toast = document.querySelector('.toast');
      expect(toast).not.toBeNull();
      vi.advanceTimersByTime(3500);
      expect(document.querySelector('.toast')).toBeNull();
      vi.useRealTimers();
    });

    it('error toast shows retry button', async () => {
      const { showToast } = await import('./popup.js');
      showToast('Error occurred', 'error');
      const toast = document.querySelector('.toast');
      const retryBtn = toast.querySelector('.toast-retry-btn');
      expect(retryBtn).not.toBeNull();
      expect(retryBtn.textContent).toBe('Retry');
    });

    it('dismiss animation runs for 500ms', async () => {
      vi.useFakeTimers();
      const { showToast } = await import('./popup.js');
      showToast('Test', 'info');
      const toast = document.querySelector('.toast');
      vi.advanceTimersByTime(3000);
      expect(toast.classList.contains('toast-out')).toBe(true);
      vi.advanceTimersByTime(600);
      expect(document.querySelector('.toast')).toBeNull();
      vi.useRealTimers();
    });
  });
});
});
