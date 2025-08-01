package oauth

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/komari-monitor/komari/database"
	"github.com/komari-monitor/komari/database/models"
	"github.com/komari-monitor/komari/utils/oauth/factory"
)

var (
	currentProvider factory.IOidcProvider
	mu              = sync.Mutex{}
	once            = sync.Once{}
)

func CurrentProvider() factory.IOidcProvider {
	mu.Lock()
	defer mu.Unlock()
	return currentProvider
}

func LoadProvider(name string, configJson string) error {
	mu.Lock()
	defer mu.Unlock()
	go func() {
		once.Do(func() {
			all := factory.GetAllOidcProviders()
			for _, provider := range all {
				if _, err := database.GetOidcConfigByName(provider.GetName()); err == nil {
					continue
				}
				// 如果数据库中没有该提供者的配置，则保存默认配置
				config := provider.GetConfiguration()
				configBytes, err := json.Marshal(config)
				if err != nil {
					log.Printf("Failed to marshal config for provider %s: %v", provider.GetName(), err)
					return
				}
				if err := database.SaveOidcConfig(&models.OidcProvider{
					Name:     provider.GetName(),
					Addition: string(configBytes),
				}); err != nil {
					log.Printf("Failed to save default config for provider %s: %v", provider.GetName(), err)
					return
				}
			}
		})
	}()
	if currentProvider != nil && currentProvider.GetName() == name {
		return nil // 已经加载了相同的提供程序
	}
	if currentProvider != nil {
		if err := currentProvider.Destroy(); err != nil {
			log.Printf("Failed to destroy provider %s: %v", currentProvider.GetName(), err)
		}
	}
	constructor, exists := factory.GetConstructor(name)
	if !exists {
		return fmt.Errorf("provider %s not found", name)
	}
	currentProvider = constructor()
	if err := json.Unmarshal([]byte(configJson), currentProvider.GetConfiguration()); err != nil {
		return fmt.Errorf("failed to unmarshal config for provider %s: %w", name, err)
	}
	err := currentProvider.Init()
	if err != nil {
		return fmt.Errorf("failed to initialize provider %s: %w", name, err)
	}
	return nil
}
