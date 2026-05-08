package hotkey

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

type callback func()

const (
	opRegister = iota
	opUnregister
	opUnregisterAll
)

const (
	WM_HOTKEY       = 0x0312
	WM_USER_COMMAND = 0x0401
)

type hotkeyCommand struct {
	op     int
	hotkey string
	cb     callback
	errCh  chan error
}

type HotkeyManager struct {
	mu         sync.RWMutex
	callbacks  map[uint32]callback
	nextID     uint32
	registered map[string]uint32
	active     atomic.Int32
	cmdCh      chan hotkeyCommand
	threadID   atomic.Uint32
}

func NewManager() *HotkeyManager {
	return &HotkeyManager{
		callbacks:  make(map[uint32]callback),
		registered: make(map[string]uint32),
		cmdCh:      make(chan hotkeyCommand, 100),
	}
}

func (m *HotkeyManager) Register(ctx context.Context, hotkey string, cb callback) error {
	if hotkey == "" {
		return errors.New("hotkey cannot be empty")
	}

	if m.active.Load() == 0 {
		return errors.New("hotkey manager is not active")
	}

	errCh := make(chan error, 1)
	m.cmdCh <- hotkeyCommand{op: opRegister, hotkey: hotkey, cb: cb, errCh: errCh}
	syscall.NewLazyDLL("user32.dll").NewProc("PostThreadMessageW").Call(uintptr(m.threadID.Load()), WM_USER_COMMAND, 0, 0)

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *HotkeyManager) Unregister(hotkey string) error {
	if m.active.Load() == 0 {
		return errors.New("hotkey manager is not active")
	}

	errCh := make(chan error, 1)
	m.cmdCh <- hotkeyCommand{op: opUnregister, hotkey: hotkey, errCh: errCh}
	syscall.NewLazyDLL("user32.dll").NewProc("PostThreadMessageW").Call(uintptr(m.threadID.Load()), WM_USER_COMMAND, 0, 0)

	return <-errCh
}

func (m *HotkeyManager) UnregisterAll() {
	if m.active.Load() == 0 {
		return
	}

	errCh := make(chan error, 1)
	m.cmdCh <- hotkeyCommand{op: opUnregisterAll, errCh: errCh}
	syscall.NewLazyDLL("user32.dll").NewProc("PostThreadMessageW").Call(uintptr(m.threadID.Load()), WM_USER_COMMAND, 0, 0)
	<-errCh
}

func (m *HotkeyManager) Update(ctx context.Context, hotkey string, cb callback) error {
	m.Unregister(hotkey)
	return m.Register(ctx, hotkey, cb)
}

func (m *HotkeyManager) Start(ctx context.Context) {
	if !m.active.CompareAndSwap(0, 1) {
		return
	}

	ready := make(chan struct{})
	go m.run(ctx, ready)
	<-ready
}

func (m *HotkeyManager) run(ctx context.Context, ready chan struct{}) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	tid, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetCurrentThreadId").Call()
	m.threadID.Store(uint32(tid))

	// Ensure thread has a message queue
	var msg Msg
	syscall.NewLazyDLL("user32.dll").NewProc("PeekMessageW").Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 0)

	close(ready)

	go func() {
		<-ctx.Done()
		syscall.NewLazyDLL("user32.dll").NewProc("PostThreadMessageW").Call(uintptr(tid), 0x0012, 0, 0) // WM_QUIT
	}()

	for {
		ret, _, _ := syscall.NewLazyDLL("user32.dll").NewProc("GetMessageW").Call(
			uintptr(unsafe.Pointer(&msg)), 0, 0, 0)

		if ret == 0 || int32(ret) == -1 {
			break
		}

		if msg.Message == WM_HOTKEY {
			id := uint32(msg.Wparam)
			m.mu.RLock()
			cb, ok := m.callbacks[id]
			m.mu.RUnlock()
			if ok {
				cb()
			}
		} else if msg.Message == WM_USER_COMMAND {
			for {
				select {
				case cmd := <-m.cmdCh:
					cmd.errCh <- m.handleCommand(cmd)
				default:
					goto nextMessage
				}
			}
		}
	nextMessage:
	}

	m.handleCommand(hotkeyCommand{op: opUnregisterAll})
	m.active.Store(0)
}

func (m *HotkeyManager) handleCommand(cmd hotkeyCommand) error {
	switch cmd.op {
	case opRegister:
		mod, key, err := parseHotkey(cmd.hotkey)
		if err != nil {
			return err
		}

		m.mu.Lock()
		id := m.nextID
		m.nextID++
		m.callbacks[id] = cmd.cb
		m.registered[cmd.hotkey] = id
		m.mu.Unlock()

		VK := uint16(key)
		if ret, _, _ := syscall.NewLazyDLL("user32.dll").NewProc("RegisterHotKey").Call(
			uintptr(0), uintptr(id), uintptr(mod), uintptr(VK)); ret == 0 {
			errno, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetLastError").Call()
			m.mu.Lock()
			delete(m.callbacks, id)
			delete(m.registered, cmd.hotkey)
			m.mu.Unlock()
			return fmt.Errorf("failed to register hotkey: errno %d", errno)
		}
		return nil

	case opUnregister:
		m.mu.Lock()
		id, ok := m.registered[cmd.hotkey]
		if !ok {
			m.mu.Unlock()
			return errors.New("hotkey not registered")
		}
		syscall.NewLazyDLL("user32.dll").NewProc("UnregisterHotKey").Call(uintptr(0), uintptr(id))
		delete(m.callbacks, id)
		delete(m.registered, cmd.hotkey)
		m.mu.Unlock()
		return nil

	case opUnregisterAll:
		m.mu.Lock()
		for _, id := range m.registered {
			syscall.NewLazyDLL("user32.dll").NewProc("UnregisterHotKey").Call(uintptr(0), uintptr(id))
		}
		m.callbacks = make(map[uint32]callback)
		m.registered = make(map[string]uint32)
		m.mu.Unlock()
		return nil
	}
	return nil
}

func (m *HotkeyManager) IsActive() bool {
	return m.active.Load() == 1
}

func (m *HotkeyManager) IsRegistered(hotkey string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.registered[hotkey]
	return ok
}

func (m *HotkeyManager) GetRegisteredHotkeys() []string {
	m.mu.RLock()
	hotkeys := make([]string, 0, len(m.registered))
	for h := range m.registered {
		hotkeys = append(hotkeys, h)
	}
	m.mu.RUnlock()
	return hotkeys
}

type Msg struct {
	HWnd   uintptr
	Message uint32
	Wparam uintptr
	Lparam uintptr
}

const (
	MOD_ALT     = 0x0001
	MOD_CONTROL = 0x0002
	MOD_SHIFT   = 0x0004
	MOD_WIN     = 0x0008
	MOD_NOREPEAT = 0x8000
)

var modifierMap = map[string]uint32{
	"Ctrl":  MOD_CONTROL,
	"Shift": MOD_SHIFT,
	"Alt":   MOD_ALT,
	"Win":   MOD_WIN,
}

var keyMap = map[string]uint16{
	"A": 0x41, "B": 0x42, "C": 0x43, "D": 0x44, "E": 0x45,
	"F": 0x46, "G": 0x47, "H": 0x48, "I": 0x49, "J": 0x4A,
	"K": 0x4B, "L": 0x4C, "M": 0x4D, "N": 0x4E, "O": 0x4F,
	"P": 0x50, "Q": 0x51, "R": 0x52, "S": 0x53, "T": 0x54,
	"U": 0x55, "V": 0x56, "W": 0x57, "X": 0x58, "Y": 0x59,
	"Z": 0x5A,
	"0": 0x30, "1": 0x31, "2": 0x32, "3": 0x33, "4": 0x34,
	"5": 0x35, "6": 0x36, "7": 0x37, "8": 0x38, "9": 0x39,
	"F1": 0x70, "F2": 0x71, "F3": 0x72, "F4": 0x73,
	"F5": 0x74, "F6": 0x75, "F7": 0x76, "F8": 0x77,
	"F9": 0x78, "F10": 0x79, "F11": 0x7A, "F12": 0x7B,
}

func parseHotkey(hotkey string) (uint32, uint16, error) {
	var mod uint32
	parts := strings.Split(hotkey, "+")
	if len(parts) < 2 {
		return 0, 0, errors.New("invalid hotkey format")
	}

	for _, part := range parts[:len(parts)-1] {
		part = strings.TrimSpace(part)
		if m, ok := modifierMap[part]; ok {
			mod |= m
		} else {
			return 0, 0, errors.New("unknown modifier: " + part)
		}
	}

	keyPart := strings.TrimSpace(parts[len(parts)-1])
	vk, ok := keyMap[strings.ToUpper(keyPart)]
	if !ok {
		return 0, 0, errors.New("unknown key: " + keyPart)
	}

	return mod, vk, nil
}