package storage

import (
	"CanMe/backend/consts"
	"CanMe/backend/pkg/logger"
	"CanMe/backend/types"
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type PreferencesStorage struct {
	storage *LocalStorage
	mutex   sync.Mutex
}

func NewPreferences() *PreferencesStorage {
	storage := NewLocalStore(consts.PREFERENCES_FILE_NAME)
	log.Printf("preferences path: %s\n", storage.ConfPath)
	return &PreferencesStorage{
		storage: storage,
	}
}

func (p *PreferencesStorage) DefaultPreferences() types.Preferences {
	return types.NewPreferences()
}

func (p *PreferencesStorage) getPreferences() (ret types.Preferences) {
	ret = p.DefaultPreferences()
	b, err := p.storage.Load()
	if err != nil {
		return
	}

	if err = yaml.Unmarshal(b, &ret); err != nil {
		ret = p.DefaultPreferences()
		return
	}

	// 如果 logger 配置为空，使用默认配置
	if reflect.DeepEqual(ret.Logger, logger.Config{}) {
		ret.Logger = *logger.DefaultConfig()
	}

	return
}

// GetPreferences Get preferences from local
func (p *PreferencesStorage) GetPreferences() (ret types.Preferences) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	ret = p.getPreferences()
	ret.Behavior.WindowWidth = max(ret.Behavior.WindowWidth, consts.MIN_WINDOW_WIDTH)
	ret.Behavior.WindowHeight = max(ret.Behavior.WindowHeight, consts.MIN_WINDOW_HEIGHT)

	if reflect.DeepEqual(ret.ListendInfo, types.ListendInfo{}) {
		ret.ListendInfo = types.DefaultListendInfo()
	}

	return
}

func (p *PreferencesStorage) setPreferences(pf *types.Preferences, key string, value any) error {
	parts := strings.Split(key, ".")
	if len(parts) > 0 {
		var reflectValue reflect.Value
		if reflect.TypeOf(pf).Kind() == reflect.Ptr {
			reflectValue = reflect.ValueOf(pf).Elem()
		} else {
			reflectValue = reflect.ValueOf(pf)
		}
		for i, part := range parts {
			part = strings.ToUpper(part[:1]) + part[1:]
			reflectValue = reflectValue.FieldByName(part)
			if reflectValue.IsValid() {
				if i == len(parts)-1 {
					reflectValue.Set(reflect.ValueOf(value))
					return nil
				}
			} else {
				break
			}
		}
	}

	return fmt.Errorf("invalid key path(%s)", key)
}

func (p *PreferencesStorage) savePreferences(pf *types.Preferences) error {
	b, err := yaml.Marshal(pf)
	if err != nil {
		return err
	}

	if err = p.storage.Store(b); err != nil {
		return err
	}

	return nil
}

// SetPreferences replace preferences
func (p *PreferencesStorage) SetPreferences(pf *types.Preferences) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return p.savePreferences(pf)
}

// UpdatePreferences update values by key paths, the key path use "." to indicate multiple level
func (p *PreferencesStorage) UpdatePreferences(values map[string]any) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	pf := p.getPreferences()
	for path, v := range values {
		if err := p.setPreferences(&pf, path, v); err != nil {
			return err
		}
	}

	return p.savePreferences(&pf)
}

func (p *PreferencesStorage) RestoreDefault() types.Preferences {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	pf := p.DefaultPreferences()
	p.savePreferences(&pf)
	return pf
}

func (p *PreferencesStorage) ConfigPath() string {
	return p.storage.ConfPath
}
