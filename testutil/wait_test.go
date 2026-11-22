package testutil

import (
	"testing"
	"time"
)

func TestPollSucceedsAfterSeveralTries(t *testing.T) {
	calls := 0
	ok := poll(time.Second, time.Millisecond, func() bool {
		calls++
		return calls == 3
	})
	if !ok || calls != 3 {
		t.Fatalf("ok=%v calls=%d, harus berhenti tepat saat kondisi terpenuhi", ok, calls)
	}
}

func TestPollTimesOut(t *testing.T) {
	start := time.Now()
	ok := poll(30*time.Millisecond, 5*time.Millisecond, func() bool { return false })
	if ok {
		t.Fatal("poll harus gagal bila kondisi tidak pernah terpenuhi")
	}
	if time.Since(start) > time.Second {
		t.Fatal("poll tidak berhenti sesuai timeout")
	}
}

func TestPollChecksOnceWithZeroTimeout(t *testing.T) {
	if !poll(0, time.Millisecond, func() bool { return true }) {
		t.Fatal("kondisi yang sudah benar harus lolos walau timeout 0")
	}
}

func TestUniqueDiffersAndKeepsPrefix(t *testing.T) {
	a, b := Unique("booking"), Unique("booking")
	if a == b {
		t.Fatalf("Unique harus berbeda tiap panggilan: %s", a)
	}
	if len(a) < 10 || a[:3] != "it-" {
		t.Fatalf("awalan data test harus it-: %s", a)
	}
}
