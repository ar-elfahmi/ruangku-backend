package queue

import "testing"

func TestLoadConfigDefaults(t *testing.T) {
	for _, key := range []string{"REDIS_ADDR", "REDIS_DB", "WORKER_CONCURRENCY"} {
		t.Setenv(key, "")
	}

	cfg := LoadConfig()

	if cfg.RedisAddr != "127.0.0.1:6379" || cfg.RedisDB != 0 {
		t.Fatalf("redis default salah: %+v", cfg)
	}
	if cfg.Queue != DefaultQueue || cfg.Concurrency != 5 {
		t.Fatalf("queue/concurrency default salah: %+v", cfg)
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("REDIS_ADDR", "10.0.0.5:6380")
	t.Setenv("REDIS_DB", "15")
	t.Setenv("WORKER_CONCURRENCY", "2")

	cfg := LoadConfig()

	if cfg.RedisAddr != "10.0.0.5:6380" || cfg.RedisDB != 15 || cfg.Concurrency != 2 {
		t.Fatalf("env tidak terbaca: %+v", cfg)
	}
}

func TestLoadConfigRejectsNonPositiveConcurrency(t *testing.T) {
	for _, value := range []string{"0", "-3"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("WORKER_CONCURRENCY", value)
			if got := LoadConfig().Concurrency; got != 5 {
				t.Fatalf("concurrency %q harus kembali ke default 5, dapat %d", value, got)
			}
		})
	}
}
