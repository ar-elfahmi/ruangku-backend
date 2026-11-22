// Package testutil berisi bantuan untuk integration test (build tag
// integration). Paket ini tidak dipakai oleh kode produksi.
package testutil

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
)

// testRedisDB adalah satu-satunya Redis logical DB yang boleh dipakai test.
const testRedisDB = 15

var (
	// ErrNotConfigured berarti variabel TEST_* belum diset; test di-skip.
	ErrNotConfigured = errors.New("variabel TEST_* belum diset")
	// ErrUnsafeTarget berarti target test bukan database test; test ditolak.
	ErrUnsafeTarget = errors.New("target test tidak aman")
)

// Env adalah konfigurasi integration test.
type Env struct {
	APIURL        string
	DBURL         string
	RedisAddr     string
	RedisDB       int
	MongoURI      string
	MongoDB       string
	AdminEmail    string
	AdminPassword string
}

// Require membaca TEST_* dari environment. Test di-skip bila belum
// dikonfigurasi dan dihentikan bila targetnya bukan database test.
func Require(t testing.TB) Env {
	t.Helper()
	env, err := LoadEnv(os.Getenv)
	switch {
	case errors.Is(err, ErrNotConfigured):
		t.Skip(err.Error())
	case err != nil:
		t.Fatalf("integration test ditolak: %v", err)
	}
	return env
}

// LoadEnv membaca dan memvalidasi TEST_*. get dikirim dari luar agar fungsi
// ini bisa dites tanpa mengubah environment proses.
func LoadEnv(get func(string) string) (Env, error) {
	values := map[string]string{}
	var missing []string
	for _, key := range []string{
		"TEST_API_URL", "TEST_DB_URL", "TEST_REDIS_ADDR", "TEST_REDIS_DB",
		"TEST_MONGO_URI", "TEST_MONGO_DB", "TEST_ADMIN_EMAIL", "TEST_ADMIN_PASSWORD",
	} {
		values[key] = strings.TrimSpace(get(key))
		if values[key] == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return Env{}, fmt.Errorf("%w: %s", ErrNotConfigured, strings.Join(missing, ", "))
	}

	redisDB, err := strconv.Atoi(values["TEST_REDIS_DB"])
	if err != nil || redisDB != testRedisDB {
		return Env{}, fmt.Errorf("%w: TEST_REDIS_DB harus %d", ErrUnsafeTarget, testRedisDB)
	}
	if !strings.HasSuffix(values["TEST_MONGO_DB"], "_test") {
		return Env{}, fmt.Errorf("%w: TEST_MONGO_DB harus berakhiran _test", ErrUnsafeTarget)
	}
	dbName, err := postgresDatabaseName(values["TEST_DB_URL"])
	if err != nil || !strings.HasSuffix(dbName, "_test") {
		return Env{}, fmt.Errorf("%w: database di TEST_DB_URL harus berakhiran _test", ErrUnsafeTarget)
	}

	return Env{
		APIURL:        strings.TrimRight(values["TEST_API_URL"], "/"),
		DBURL:         values["TEST_DB_URL"],
		RedisAddr:     values["TEST_REDIS_ADDR"],
		RedisDB:       redisDB,
		MongoURI:      values["TEST_MONGO_URI"],
		MongoDB:       values["TEST_MONGO_DB"],
		AdminEmail:    values["TEST_ADMIN_EMAIL"],
		AdminPassword: values["TEST_ADMIN_PASSWORD"],
	}, nil
}

func postgresDatabaseName(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return strings.Trim(u.Path, "/"), nil
}
