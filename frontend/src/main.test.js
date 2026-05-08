import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

const mockIsOverlay = vi.fn();
const mockHideOverlay = vi.fn();
const mockShowOverlay = vi.fn();
const mockWindowShow = vi.fn();
const mockWindowHide = vi.fn();
const mockEventsOn = vi.fn();
const mockShowToast = vi.fn();

vi.mock('../wailsjs/go/main/App', () => ({
  ShowOverlay: mockShowOverlay,
  HideOverlay: mockHideOverlay,
  IsOverlay: mockIsOverlay,
  ToggleOverlay: vi.fn(),
  GetPinnedMessages: vi.fn().mockResolvedValue([]),
  ResizeWindow: vi.fn().mockResolvedValue(null),
  SaveCurrentPosition: vi.fn().mockResolvedValue(null),
}));

vi.mock('../wailsjs/runtime', () => ({
  WindowShow: mockWindowShow,
  WindowHide: mockWindowHide,
  EventsOn: mockEventsOn,
}));

vi.mock('./popup.js', () => ({
  showPopup: vi.fn(),
  hidePopup: vi.fn(),
  toggleSettings: vi.fn(),
  pinCurrent: vi.fn(),
  showToast: mockShowToast,
  updateQueue: vi.fn(),
  resetModuleState: vi.fn(),
}));

describe('toggleOverlay', () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    mockIsOverlay.mockResolvedValue(false);
    mockHideOverlay.mockResolvedValue();
    mockShowOverlay.mockResolvedValue();
    mockWindowShow.mockResolvedValue();
    mockWindowHide.mockResolvedValue();
    document.body.innerHTML = '<div id="app"></div>';
    const { resetModuleState } = await import('./popup.js');
    resetModuleState();
  });

  it('shows overlay and calls WindowShow when overlay is hidden', async () => {
    mockIsOverlay.mockResolvedValue(false);
    const { toggleOverlay } = await import('../src/main.js');
    await toggleOverlay();
    expect(mockShowOverlay).toHaveBeenCalled();
    expect(mockWindowShow).toHaveBeenCalled();
    expect(mockHideOverlay).not.toHaveBeenCalled();
    expect(mockWindowHide).not.toHaveBeenCalled();
  });

  it('hides overlay and calls WindowHide when overlay is visible', async () => {
    mockIsOverlay.mockResolvedValue(true);
    const { toggleOverlay } = await import('../src/main.js');
    await toggleOverlay();
    expect(mockHideOverlay).toHaveBeenCalled();
    expect(mockWindowHide).toHaveBeenCalled();
    expect(mockShowOverlay).not.toHaveBeenCalled();
    expect(mockWindowShow).not.toHaveBeenCalled();
  });
});

describe('Escape key handling', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockIsOverlay.mockResolvedValue(false);
    mockHideOverlay.mockResolvedValue();
    mockWindowHide.mockResolvedValue();
  });

  it('hides overlay and window when Escape is pressed', async () => {
    await import('../src/main.js');
    const event = new KeyboardEvent('keydown', { key: 'Escape' });
    document.dispatchEvent(event);
    await new Promise(setImmediate);
    expect(mockHideOverlay).toHaveBeenCalled();
    expect(mockWindowHide).toHaveBeenCalled();
  });
});

describe('Service event toasts', () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    document.body.innerHTML = '<div id="app"></div>';
    const { resetModuleState } = await import('./popup.js');
    resetModuleState();
  });

  it('shows success toast on service-connected event', async () => {
    const { initServiceEventHandlers } = await import('../src/main.js');
    initServiceEventHandlers();
    const serviceConnectedHandler = mockEventsOn.mock.calls.find(call => call[0] === 'service-connected')[1];
    serviceConnectedHandler({ service: 'twitch' });
    expect(mockShowToast).toHaveBeenCalledWith('Connected to twitch', 'success');
  });

  it('shows error toast on service-disconnected event', async () => {
    const { initServiceEventHandlers } = await import('../src/main.js');
    initServiceEventHandlers();
    const handler = mockEventsOn.mock.calls.find(call => call[0] === 'service-disconnected')[1];
    handler({ service: 'twitch' });
    expect(mockShowToast).toHaveBeenCalledWith('Disconnected from twitch', 'error');
  });

  it('shows info toast on service-reconnecting event with attempt count', async () => {
    const { initServiceEventHandlers } = await import('../src/main.js');
    initServiceEventHandlers();
    const handler = mockEventsOn.mock.calls.find(call => call[0] === 'service-reconnecting')[1];
    handler({ service: 'twitch', attempt: 3, maxAttempts: 5 });
    expect(mockShowToast).toHaveBeenCalledWith('Reconnecting to twitch (attempt 3/5)', 'info');
  });
});