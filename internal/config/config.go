package config

import (
	"fmt"
	"os"
	"scheduler/pkg/lib/logger/zaplogger"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func LoadServiceConfig(log *zap.Logger, configPath, dbPasswordPath string) (ServiceConfig, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigFile(configPath)
	if err := v.ReadInConfig(); err != nil {
		log.Error("Failed to Read config", zaplogger.Err(err))
		return ServiceConfig{}, err
	}

	var serviceConfig ServiceConfig
	if err := v.Unmarshal(&serviceConfig); err != nil {
		log.Error("Failed to Unmarshal config", zaplogger.Err(err))
		return ServiceConfig{}, err
	}
	dbConnStr, err := serviceConfig.DSN(log, dbPasswordPath)
	if err != nil {
		log.Error("Error generating DSN for database connection", zaplogger.Err(err))
		return ServiceConfig{}, err
	}
	serviceConfig.DbConfig.DBConn = dbConnStr
	log.Info("Config", zap.Any("serviceConfig", serviceConfig))
	return serviceConfig, nil
}

func (d ServiceConfig) DSN(log *zap.Logger, dbPasswordPath string) (string, error) {
	password := os.Getenv(dbPasswordPath)
	if password == "" {
		return "", fmt.Errorf("environment variable %s is not set", dbPasswordPath)
	}

	return fmt.Sprintf("%s://%s:%s@%s:%d/%s",
		d.DbConfig.Driver, d.DbConfig.User, password, d.DbConfig.Host, d.DbConfig.Port, d.DbConfig.DBName), nil
}

type ServiceConfig struct {
	Address    string     `yaml:"address"`
	DbConfig   DBConfig   `mapstructure:"database"`
	NATSConfig NATSConfig `mapstructure:"nats"`
	// RedisConfig redis.RedisConfig `mapstructure:"redis_config"`
	// KafkaConfig kafka.KafkaConfig `mapstructure:"kafka_config"`
}
type DBConfig struct {
	Driver string `yaml:"driver"`
	Host   string `yaml:"host"`
	Port   int    `yaml:"port"`
	User   string `yaml:"user"`
	DBName string `yaml:"dbname"`
	DBConn string
}

type NATSConfig struct {
	URL string `yaml:"url"`
}
