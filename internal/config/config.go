package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Port        string `mapstructure:"PORT"`
	DatabaseURL string `mapstructure:"DATABASE_URL"`
	JWTSecret   string `mapstructure:"JWT_SECRET"`
	RedisURL    string `mapstructure:"REDIS_URL"`
}

func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	if _, err := os.Stat(".env"); err == nil {
		err := godotenv.Load()
		if err != nil {
			log.Printf("Error loading .env file: %v", err)
		}
	}

	v := viper.New()
	v.AutomaticEnv() // Read environment variables

	// Set default values
	v.SetDefault("PORT", "8080")
	v.SetDefault("DATABASE_URL", "postgres://user:password@localhost:5432/db?sslmode=disable")
	v.SetDefault("JWT_SECRET", "supersecretjwtkey")
	v.SetDefault("REDIS_URL", "redis://localhost:6379/0")

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
