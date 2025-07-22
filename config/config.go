package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port       string
	DbHost     string
	DbPort     string
	DbUser     string
	DbPass     string
	DbName     string
	DateFormat string
}

func LoadConfig() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}

	return &Config{
		Port:       os.Getenv("PORT"),
		DbHost:     os.Getenv("DB_HOST"),
		DbPort:     os.Getenv("DB_PORT"),
		DbUser:     os.Getenv("DB_USER"),
		DbPass:     os.Getenv("DB_PASSWORD"),
		DbName:     os.Getenv("DB_NAME"),
		DateFormat: os.Getenv("DATE_FORMAT"),
	}, nil
}
