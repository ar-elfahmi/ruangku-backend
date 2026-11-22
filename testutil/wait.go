package testutil

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// Batas tunggu standar untuk test asynchronous (worker memproses task).
const (
	DefaultWait     = 10 * time.Second
	DefaultInterval = 200 * time.Millisecond
)

var uniqueCounter atomic.Int64

// WaitFor menunggu cond menjadi true. Test asynchronous memakai ini, bukan
// time.Sleep dengan durasi tetap.
func WaitFor(t testing.TB, timeout, interval time.Duration, cond func() bool, what string) {
	t.Helper()
	if !poll(timeout, interval, cond) {
		t.Fatalf("timeout %s menunggu: %s", timeout, what)
	}
}

// poll memeriksa cond minimal sekali, lalu berulang tiap interval sampai
// timeout. Mengembalikan true bila cond terpenuhi.
func poll(timeout, interval time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	for {
		if cond() {
			return true
		}
		if !time.Now().Before(deadline) {
			return false
		}
		time.Sleep(interval)
	}
}

// Unique membuat nama data test yang tidak bentrok (awalan it-, label, waktu,
// penghitung) supaya test tidak perlu menghapus data apa pun.
func Unique(label string) string {
	return fmt.Sprintf("it-%s-%d-%d", label, time.Now().UnixNano(), uniqueCounter.Add(1))
}
