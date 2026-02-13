package config

import "os"

type Config struct {
	Addr      string
	DBPath    string
	JWTSecret string
}

func FromEnv() Config {
	cfg := Config{
		Addr:      getEnv("SERVER_ADDR", ":6365"),
		DBPath:    getEnv("DB_PATH", "/app/data/gost.db"),
		JWTSecret: getEnv("JWT_SECRET", ""),
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
