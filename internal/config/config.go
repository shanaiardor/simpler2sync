package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/ini.v1"
)

type R2Config struct {
	Type            string `json:"type"`
	Provider        string `json:"provider"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Endpoint        string `json:"endpoint"`
	Region          string `json:"region"`
	ACL             string `json:"acl"`
}

type SyncTask struct {
	Name          string `json:"name"`
	LocalPath     string `json:"local_path"`
	RemoteBucket  string `json:"remote_bucket"`
	RemotePrefix  string `json:"remote_prefix"`
	Enabled       bool   `json:"enabled"`
}

type AppConfig struct {
	R2        R2Config   `json:"r2"`
	Tasks     []SyncTask `json:"sync_tasks"`
	Settings  Settings   `json:"settings"`
	configDir string     `json:"-"`
}

type Settings struct {
	IntervalSeconds     int    `json:"interval_seconds"`
	CronExpression      string `json:"cron_expression"`
	ConflictStrategy    string `json:"conflict_strategy"`
	ConcurrentTransfers int    `json:"concurrent_transfers"`
}

func DefaultSettings() Settings {
	return Settings{
		IntervalSeconds:     300,
		ConflictStrategy:    "newer",
		ConcurrentTransfers: 3,
	}
}

func configDir() string {
	d, err := os.UserConfigDir()
	if err != nil || d == "" {
		d, _ = os.Getwd()
	}
	return filepath.Join(d, "simpler2sync")
}

func ConfigPath() string {
	return filepath.Join(configDir(), "config.json")
}

func ConfigPathDir() string {
	return configDir()
}

func Load() (*AppConfig, error) {
	cfg := &AppConfig{
		configDir: configDir(),
		Settings:  DefaultSettings(),
	}
	if err := os.MkdirAll(cfg.configDir, 0700); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}
	data, err := os.ReadFile(ConfigPath())
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

func (c *AppConfig) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(ConfigPath(), data, 0600)
}

func LoadR2FromINIPath(path string) (*R2Config, error) {
	f, err := ini.Load(path)
	if err != nil {
		return nil, fmt.Errorf("load ini: %w", err)
	}
	sec := f.Section("r2")
	cfg := &R2Config{
		Type:            sec.Key("type").String(),
		Provider:        sec.Key("provider").String(),
		AccessKeyID:     sec.Key("access_key_id").String(),
		SecretAccessKey: sec.Key("secret_access_key").String(),
		Endpoint:        sec.Key("endpoint").String(),
		Region:          sec.Key("region").String(),
		ACL:             sec.Key("acl").String(),
	}
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("missing endpoint in [r2] section")
	}
	return cfg, nil
}
