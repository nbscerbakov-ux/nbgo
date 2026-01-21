package conf

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

// ConfigManager manages all configurations for NBGO
type ConfigManager struct {
	mu           sync.RWMutex
	config       map[string]interface{}
	validators   map[string]ConfigValidator
	watchers     map[string][]ConfigWatcher
	lastModified time.Time
	configPath   string
	backupDir    string
	logger       io.Writer
}

// ConfigValidator validates configuration
type ConfigValidator interface {
	Validate(config interface{}) error
}

// ConfigWatcher watches for configuration changes
type ConfigWatcher interface {
	OnChange(key string, oldValue, newValue interface{}) error
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(configPath string) *ConfigManager {
	return &ConfigManager{
		config:       make(map[string]interface{}),
		validators:   make(map[string]ConfigValidator),
		watchers:     make(map[string][]ConfigWatcher),
		configPath:   configPath,
		backupDir:    "./config-backups",
		logger:       os.Stdout,
		lastModified: time.Now(),
	}
}

// SetLogger sets the logger for the config manager
func (cm *ConfigManager) SetLogger(logger io.Writer) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.logger = logger
}

// log logs a message
func (cm *ConfigManager) log(format string, args ...interface{}) {
	if cm.logger != nil {
		fmt.Fprintf(cm.logger, "[ConfigManager] "+format+"\n", args...)
	}
}

// RegisterValidator registers a validator for a config key
func (cm *ConfigManager) RegisterValidator(key string, validator ConfigValidator) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.validators[key] = validator
}

// RegisterWatcher registers a watcher for a config key
func (cm *ConfigManager) RegisterWatcher(key string, watcher ConfigWatcher) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if _, exists := cm.watchers[key]; !exists {
		cm.watchers[key] = make([]ConfigWatcher, 0)
	}
	cm.watchers[key] = append(cm.watchers[key], watcher)
}

// Set sets a configuration value
func (cm *ConfigManager) Set(key string, value interface{}) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Validate if validator exists
	if validator, exists := cm.validators[key]; exists {
		if err := validator.Validate(value); err != nil {
			cm.log("Validation failed for key %s: %v", key, err)
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	oldValue, _ := cm.config[key]
	cm.config[key] = value
	cm.lastModified = time.Now()
	cm.log("Set %s = %v", key, value)

	// Notify watchers
	if watcherList, exists := cm.watchers[key]; exists {
		for _, watcher := range watcherList {
			if err := watcher.OnChange(key, oldValue, value); err != nil {
				cm.log("Watcher error for key %s: %v", key, err)
			}
		}
	}

	return nil
}

// Get retrieves a configuration value
func (cm *ConfigManager) Get(key string) (interface{}, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	value, exists := cm.config[key]
	return value, exists
}

// GetString retrieves a string configuration value
func (cm *ConfigManager) GetString(key string) (string, bool) {
	value, exists := cm.Get(key)
	if !exists {
		return "", false
	}
	strValue, ok := value.(string)
	return strValue, ok
}

// GetInt retrieves an integer configuration value
func (cm *ConfigManager) GetInt(key string) (int, bool) {
	value, exists := cm.Get(key)
	if !exists {
		return 0, false
	}
	switch v := value.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// GetBool retrieves a boolean configuration value
func (cm *ConfigManager) GetBool(key string) (bool, bool) {
	value, exists := cm.Get(key)
	if !exists {
		return false, false
	}
	boolValue, ok := value.(bool)
	return boolValue, ok
}

// List returns all configuration keys
func (cm *ConfigManager) List() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	result := make(map[string]interface{})
	for k, v := range cm.config {
		result[k] = v
	}
	return result
}

// SaveJSON saves configuration to JSON file
func (cm *ConfigManager) SaveJSON(filePath string) error {
	cm.mu.RLock()
	config := make(map[string]interface{})
	for k, v := range cm.config {
		config[k] = v
	}
	cm.mu.RUnlock()

	// Create backup
	if err := cm.createBackup(filePath); err != nil {
		cm.log("Warning: failed to create backup: %v", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("write error: %w", err)
	}

	cm.log("Configuration saved to %s", filePath)
	return nil
}

// LoadJSON loads configuration from JSON file
func (cm *ConfigManager) LoadJSON(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read error: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("unmarshal error: %w", err)
	}

	cm.mu.Lock()
	cm.config = config
	cm.lastModified = time.Now()
	cm.mu.Unlock()

	cm.log("Configuration loaded from %s", filePath)
	return nil
}

// LoadYAML loads configuration from YAML file
func (cm *ConfigManager) LoadYAML(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read error: %w", err)
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("unmarshal error: %w", err)
	}

	cm.mu.Lock()
	cm.config = config
	cm.lastModified = time.Now()
	cm.mu.Unlock()

	cm.log("Configuration loaded from %s", filePath)
	return nil
}

// Backup creates a backup of the current configuration
func (cm *ConfigManager) Backup() error {
	return cm.createBackup(cm.configPath)
}

// createBackup creates a backup of configuration
func (cm *ConfigManager) createBackup(filePath string) error {
	if _, err := os.Stat(cm.backupDir); os.IsNotExist(err) {
		if err := os.MkdirAll(cm.backupDir, 0755); err != nil {
			return err
		}
	}

	timestamp := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s/config_%s.json", cm.backupDir, timestamp)

	cm.mu.RLock()
	config := make(map[string]interface{})
	for k, v := range cm.config {
		config[k] = v
	}
	cm.mu.RUnlock()

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(backupPath, data, 0644)
}

// RestoreFromBackup restores configuration from a backup
func (cm *ConfigManager) RestoreFromBackup(backupPath string) error {
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("read backup error: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("unmarshal backup error: %w", err)
	}

	cm.mu.Lock()
	cm.config = config
	cm.lastModified = time.Now()
	cm.mu.Unlock()

	cm.log("Configuration restored from %s", backupPath)
	return nil
}

// Diff returns differences between current config and another
func (cm *ConfigManager) Diff(other map[string]interface{}) map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	diff := make(map[string]interface{})
	for k, v := range cm.config {
		if otherValue, exists := other[k]; !exists || otherValue != v {
			diff[k] = v
		}
	}

	return diff
}

// Export exports configuration as JSON bytes
func (cm *ConfigManager) Export() ([]byte, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	config := make(map[string]interface{})
	for k, v := range cm.config {
		config[k] = v
	}

	return json.MarshalIndent(config, "", "  ")
}

// Import imports configuration from JSON bytes
func (cm *ConfigManager) Import(data []byte) error {
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("unmarshal error: %w", err)
	}

	cm.mu.Lock()
	cm.config = config
	cm.lastModified = time.Now()
	cm.mu.Unlock()

	return nil
}

// Monitor watches for external configuration changes
type Monitor struct {
	manager  *ConfigManager
	filePath string
	ticker   *time.Ticker
	done     chan bool
	mu       sync.Mutex
}

// NewMonitor creates a new configuration monitor
func NewMonitor(manager *ConfigManager, filePath string, interval time.Duration) *Monitor {
	return &Monitor{
		manager:  manager,
		filePath: filePath,
		ticker:   time.NewTicker(interval),
		done:     make(chan bool),
	}
}

// Start starts monitoring
func (m *Monitor) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				m.Stop()
				return
			case <-m.done:
				return
			case <-m.ticker.C:
				if err := m.manager.LoadJSON(m.filePath); err != nil {
					log.Printf("Monitor error: %v", err)
				}
			}
		}
	}()
}

// Stop stops monitoring
func (m *Monitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ticker.Stop()
	select {
	case m.done <- true:
	default:
	}
}
