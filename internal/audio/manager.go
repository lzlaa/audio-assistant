package audio

import (
	"fmt"
	"sync"

	"github.com/gordonklaus/portaudio"
)

var (
	audioManager *Manager
	managerOnce  sync.Once
)

// Manager 管理音频系统的初始化和终止
type Manager struct {
	mu          sync.Mutex
	initialized bool
	refCount    int
}

// GetManager 获取全局音频管理器实例
func GetManager() *Manager {
	managerOnce.Do(func() {
		audioManager = &Manager{}
	})
	return audioManager
}

// Initialize 初始化音频系统
func (m *Manager) Initialize() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized {
		if err := portaudio.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize PortAudio: %w", err)
		}
		m.initialized = true
	}

	m.refCount++
	return nil
}

// Terminate 终止音频系统
func (m *Manager) Terminate() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.refCount > 0 {
		m.refCount--
	}

	if m.refCount == 0 && m.initialized {
		if err := portaudio.Terminate(); err != nil {
			return fmt.Errorf("failed to terminate PortAudio: %w", err)
		}
		m.initialized = false
	}

	return nil
}

// IsInitialized 检查音频系统是否已初始化
func (m *Manager) IsInitialized() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.initialized
}
