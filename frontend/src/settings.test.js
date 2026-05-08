import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../wailsjs/go/main/App', () => ({
  GetConfig: vi.fn(),
  SaveConfig: vi.fn(),
  GetVoices: vi.fn(),
  GetPinnedMessages: vi.fn().mockResolvedValue([]),
  ResizeWindow: vi.fn().mockResolvedValue(null),
  EnterSettingsMode: vi.fn().mockResolvedValue(null),
  CancelSettings: vi.fn().mockResolvedValue(null),
}));

describe('settings panel', () => {
  let GetConfig, SaveConfig, GetVoices;

  beforeEach(async () => {
    vi.clearAllMocks();
    const appMock = await import('../wailsjs/go/main/App');
    GetConfig = appMock.GetConfig;
    SaveConfig = appMock.SaveConfig;
    GetVoices = appMock.GetVoices;

    document.body.innerHTML = '<div id="app"></div>';
    const { resetModuleState } = await import('./popup.js');
    resetModuleState();
  });

  it('loads config and populates fields on toggle', async () => {
    const config = {
      twitchOAuthToken: 'token123',
      thaiVoiceName: 'Thai1',
      englishVoiceName: 'Eng1',
      speechRateMultiplier: 1.2,
      autoFadeDelay: 10,
      maxQueueSize: 200,
      cloudTTSAPIKey: 'api-key-123',
    };
    GetConfig.mockResolvedValue(config);
    GetVoices.mockResolvedValue(['Thai1', 'Thai2', 'Eng1', 'Eng2']);

    const { toggleSettings } = await import('./popup.js');
    await toggleSettings();

    expect(GetConfig).toHaveBeenCalled();
    expect(GetVoices).toHaveBeenCalledWith('th');
    expect(GetVoices).toHaveBeenCalledWith('en');

    const panel = document.querySelector('.settings-panel');
    expect(panel).not.toBeNull();
    expect(panel.querySelector('#twitch-token').value).toBe('token123');
    expect(panel.querySelector('#speech-rate').value).toBe('1.2');
    expect(panel.querySelector('#auto-fade').value).toBe('10');
    expect(panel.querySelector('#max-queue').value).toBe('200');
    expect(panel.querySelector('#cloud-api-key').value).toBe('api-key-123');
  });

  it('switches tabs and shows hotkeys', async () => {
    const config = {
      toggleOverlayHotkey: 'Ctrl+1',
      pinLastMessageHotkey: 'Ctrl+2',
    };
    GetConfig.mockResolvedValue(config);
    GetVoices.mockResolvedValue([]);

    const { toggleSettings } = await import('./popup.js');
    await toggleSettings();

    const panel = document.querySelector('.settings-panel');
    const hotkeyTabBtn = panel.querySelector('.tab-btn[data-tab="hotkeys"]');
    const configTab = panel.querySelector('#tab-config');
    const hotkeyTab = panel.querySelector('#tab-hotkeys');

    expect(configTab.classList.contains('active')).toBe(true);
    expect(hotkeyTab.classList.contains('active')).toBe(false);

    hotkeyTabBtn.click();

    expect(configTab.classList.contains('active')).toBe(false);
    expect(hotkeyTab.classList.contains('active')).toBe(true);

    const values = Array.from(hotkeyTab.querySelectorAll('.hotkey-value')).map(el => el.textContent);
    expect(values).toContain('Ctrl+1');
    expect(values).toContain('Ctrl+2');
  });

  it('saves config when Save button is clicked', async () => {
    GetConfig.mockResolvedValue({});
    GetVoices.mockResolvedValue([]);
    SaveConfig.mockResolvedValue(null);

    const { toggleSettings } = await import('./popup.js');
    await toggleSettings();

    const panel = document.querySelector('.settings-panel');
    panel.querySelector('#twitch-token').value = 'new-token';
    panel.querySelector('#save-settings').click();

    expect(SaveConfig).toHaveBeenCalledWith(expect.objectContaining({
      twitchOAuthToken: 'new-token'
    }));
  });

  it('removes panel without saving when Cancel is clicked', async () => {
    GetConfig.mockResolvedValue({});
    GetVoices.mockResolvedValue([]);

    const { toggleSettings } = await import('./popup.js');
    await toggleSettings();

    const panel = document.querySelector('.settings-panel');
    panel.querySelector('#cancel-settings').click();

    expect(SaveConfig).not.toHaveBeenCalled();
    expect(document.querySelector('.settings-panel')).toBeNull();
  });
});
