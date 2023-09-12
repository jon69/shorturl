// Модуль config представляет класс для разбора конфигурации.
package config

import (
	"encoding/json"
	"log"
	"os"
)

// ConfigHandler определяет класс управления конфигурацией.
type ConfigHandler struct {
	params configParams
}

// Parse открывает файл конфигурации и парсит его содержимое.
func Parse(configPath string) (ConfigHandler, error) {
	cfg := ConfigHandler{}
	log.Printf("opening config file %s", configPath)
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("can not open file | %s", err.Error())
		return cfg, err
	}
	err = json.Unmarshal(data, &cfg.params)
	if err != nil {
		log.Printf("can not Unmarshal json | %s", err.Error())
		return cfg, err
	}
	return cfg, nil
}

// ServerAddress возвращает адрес сервера.
func (h *ConfigHandler) ServerAddress(serverAddress string) string {
	if serverAddress != "" {
		return serverAddress
	}
	return h.params.Addr
}

// BaseURL возвращает базовый URL.
func (h *ConfigHandler) BaseURL(baseURL string) string {
	if baseURL != "" {
		return baseURL
	}
	return h.params.BaseURL
}

// FilePath возвращает путь к файлу с адресами.
func (h *ConfigHandler) FilePath(filePath string) string {
	if filePath != "" {
		return filePath
	}
	return h.params.FileStoragePath
}

// DatabaseDNS возвращает адрес БД.
func (h *ConfigHandler) DatabaseDNS(conndb string) string {
	if conndb != "" {
		return conndb
	}
	return h.params.DatabaseDNS
}

// EnableHTTPS возвращает признак использования https.
func (h *ConfigHandler) EnableHTTPS(enableHTTPS string) string {
	if enableHTTPS != "" {
		return enableHTTPS
	}
	if h.params.EnableHTTPS {
		return "true"
	}
	return ""
}

// configParams храние информацию о парамтрах конфигурации.
type configParams struct {
	// server_address - адрес сервера.
	Addr string `json:"server_address"`
	// base_url - базовый URL.
	BaseURL string `json:"base_url"`
	// file_storage_path - путь к файлу с адресами.
	FileStoragePath string `json:"file_storage_path"`
	// database_dsn - адрес БД.
	DatabaseDNS string `json:"database_dsn"`
	// enable_https - признак использования https.
	EnableHTTPS bool `json:"enable_https"`
}
