package config

import (
	"scheduler/worker/pkg/lib/logger/zaplogger"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Config struct {
	NATSURL string `mapstructure:"nats_url"`
}

func LoadConfig(log *zap.Logger, path string) (Config, error) {
	var config Config

	// Инициализируем viper
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		log.Error("Failed to read config file", zaplogger.Err(err), zap.String("path", path))
	} else {
		log.Info("Config file loaded", zap.String("path", path))
	}
	if err := v.Unmarshal(&config); err != nil {
		log.Error("Failed to unmarshal config file", zaplogger.Err(err))
		// Продолжаем работу, даже если не удалось распарсить файл
	}
	log.Info("config", zap.Any("config", config))

	return config, nil
}
