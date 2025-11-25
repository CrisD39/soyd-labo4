// internal/config/config.go
package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Logger struct {
		ListenAddr string `json:"listenAddr"`
	} `json:"logger"`

	Namenode struct {
		HTTPAddr   string `json:"httpAddr"`
		TCPAddr    string `json:"tcpAddr"`
		LoggerAddr string `json:"loggerAddr"`
	} `json:"namenode"`

	Client struct {
		NamenodeTCPAddr string `json:"namenodeTcpAddr"`
		NamenodeHTTPAddr string `json:"namenodeHttpAddr"`
		LoggerAddr       string `json:"loggerAddr"`
	} `json:"client"`

	Datanodes []struct {
		ID          string `json:"id"`
		ListenAddr  string `json:"listenAddr"`
		LoggerAddr  string `json:"loggerAddr"`
		StoragePath string `json:"storagePath"`
	} `json:"datanodes"`
}

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
