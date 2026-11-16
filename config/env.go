package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// LoadEnv memuat file .env bila ada. Di production variabel biasanya
// sudah diset oleh environment, jadi file yang hilang bukan error.
func LoadEnv() {
	_ = godotenv.Load()
}

// GetEnv membaca variabel environment, atau fallback bila kosong.
func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

// GetEnvInt membaca variabel environment sebagai integer, atau fallback
// bila kosong atau bukan angka.
func GetEnvInt(key string, fallback int) int {
	value, err := strconv.Atoi(GetEnv(key, ""))
	if err != nil {
		return fallback
	}
	return value
}
