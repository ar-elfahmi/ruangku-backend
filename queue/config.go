package queue

import (
	"github.com/hibiken/asynq"

	"ruangku-backend/config"
)

const (
	// DefaultQueue adalah nama antrean untuk task notifikasi.
	DefaultQueue = "notifications"

	defaultRedisAddr   = "127.0.0.1:6379"
	defaultConcurrency = 5
)

// Config dipakai bersama oleh producer (API) dan worker. Test memakai Queue
// yang unik supaya tidak bentrok dengan worker lain di Redis yang sama.
type Config struct {
	RedisAddr   string
	RedisDB     int
	Queue       string
	Concurrency int // hanya dibaca worker

	// RetryDelay mengganti jeda antar-retry worker. Nil memakai backoff
	// eksponensial bawaan Asynq (jeda pertama puluhan detik); hanya test yang
	// perlu menggantinya.
	RetryDelay asynq.RetryDelayFunc
}

// LoadConfig membaca REDIS_ADDR, REDIS_DB, dan WORKER_CONCURRENCY lewat
// config.GetEnv. Concurrency yang nol atau negatif kembali ke default.
func LoadConfig() Config {
	concurrency := config.GetEnvInt("WORKER_CONCURRENCY", defaultConcurrency)
	if concurrency < 1 {
		concurrency = defaultConcurrency
	}
	return Config{
		RedisAddr:   config.GetEnv("REDIS_ADDR", defaultRedisAddr),
		RedisDB:     config.GetEnvInt("REDIS_DB", 0),
		Queue:       DefaultQueue,
		Concurrency: concurrency,
	}
}

// RedisOpt mengubah Config menjadi opsi koneksi Asynq.
func (c Config) RedisOpt() asynq.RedisClientOpt {
	return asynq.RedisClientOpt{Addr: c.RedisAddr, DB: c.RedisDB}
}
