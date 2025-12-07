package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
	} `mapstructure:"server"`
	Storage struct {
		UploadDir string `mapstructure:"upload_dir"`
		BaseURL   string `mapstructure:"base_url"`
	} `mapstructure:"storage"`
	Auth struct {
		Enabled   bool   `mapstructure:"enabled"`
		SecretKey string `mapstructure:"secret_key"`
	} `mapstructure:"auth"`
}

var Conf Config

// 加载配置
func LoadConfig() error {
	var err error
	// 设置默认值
	setConfigDefaults()

	// 配置文件设置
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// 支持环境变量覆盖（可选）
	viper.SetEnvPrefix("APP")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 读取配置文件（如果不存在则使用默认值）
	if err = viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("Config file not found, using defaults and creating config.json...")
			if err := viper.SafeWriteConfigAs("config.json"); err != nil {
				return fmt.Errorf("failed to create config file: %v", err)
			}
		} else {
			return fmt.Errorf("failed to read config: %v", err)
		}
	}

	// 解析配置到结构体
	if err = viper.Unmarshal(&Conf); err != nil {
		return fmt.Errorf("failed to unmarshal config: %v", err)
	}

	// 后处理
	Conf.Storage.BaseURL = strings.TrimSuffix(Conf.Storage.BaseURL, "/")
	// 将相对路径改成绝对路径
	if Conf.Storage.UploadDir, err = filepath.Abs(Conf.Storage.UploadDir); err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	// 验证配置
	if err := validateConfig(); err != nil {
		return fmt.Errorf("invalid config: %v", err)
	}

	return nil
}

// 设置配置默认值
func setConfigDefaults() {
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 36677)
	viper.SetDefault("storage.upload_dir", "./uploads")
	viper.SetDefault("storage.base_url", "https://uploads.example.com")
	viper.SetDefault("auth.enabled", false)
	viper.SetDefault("auth.secret_key", "")
}

// 验证配置
func validateConfig() error {
	if Conf.Server.Port < 1 || Conf.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d", Conf.Server.Port)
	}

	if Conf.Storage.UploadDir == "" {
		return fmt.Errorf("upload_dir cannot be empty")
	}

	if Conf.Storage.BaseURL == "" {
		return fmt.Errorf("base_url cannot be empty")
	}

	if Conf.Auth.Enabled && Conf.Auth.SecretKey == "" {
		return fmt.Errorf("secret_key cannot be empty when auth is enabled")
	}

	return nil
}
