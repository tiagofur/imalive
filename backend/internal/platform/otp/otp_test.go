package otp

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

const testSecret = "test-hmac-secret"

func TestGenerate(t *testing.T) {
	seen := map[string]bool{}
	for range 50 {
		code, err := Generate()
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		if len(code) != CodeLength {
			t.Fatalf("code %q has length %d, want %d", code, len(code), CodeLength)
		}
		for _, r := range code {
			if r < '0' || r > '9' {
				t.Fatalf("code %q has a non-digit %q", code, r)
			}
		}
		seen[code] = true
	}
	if len(seen) < 40 {
		t.Fatalf("only %d distinct codes in 50 draws, want broad randomness", len(seen))
	}
}

func TestHashIsDeterministicAndPhoneBound(t *testing.T) {
	a := Hash(testSecret, "+5215511111111", "123456")
	b := Hash(testSecret, "+5215511111111", "123456")
	if a != b || len(a) != 64 {
		t.Fatalf("Hash not deterministic or wrong length: %q vs %q", a, b)
	}
	if a == Hash(testSecret, "+5215511111112", "123456") {
		t.Fatal("same code for a different phone produced the same hash")
	}
	if a == Hash("other-secret", "+5215511111111", "123456") {
		t.Fatal("same code with a different secret produced the same hash")
	}
	if !Verify(testSecret, "+5215511111111", "123456", a) {
		t.Fatal("Verify rejected its own hash")
	}
}

func TestStoreLifecycle(t *testing.T) {
	mr := miniredis.RunT(t)
	store := NewStore(redis.NewClient(&redis.Options{Addr: mr.Addr()}), testSecret)
	ctx := context.Background()
	phone := "+5215511111111"

	if out, err := store.Consume(ctx, phone, "123456"); err != nil || out != Expired {
		t.Fatalf("Consume before any issue = (%v, %v), want Expired/nil", out, err)
	}

	code, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if err := store.Issue(ctx, phone, code, DefaultTTL); err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if got, _ := mr.Get(key(phone)); got == code {
		t.Fatal("the code is stored in clear text in Redis")
	}
	if got, _ := mr.Get(key(phone)); got != Hash(testSecret, phone, code) {
		t.Fatal("the stored value is not the HMAC of the code")
	}

	// A wrong code consumes the attempt (single-attempt rule): the store
	// reports Invalid, and any further attempt finds nothing (Expired).
	if out, err := store.Consume(ctx, phone, "000000"); err != nil || out != Invalid {
		t.Fatalf("wrong code = (%v, %v), want Invalid/nil", out, err)
	}
	if out, err := store.Consume(ctx, phone, code); err != nil || out != Expired {
		t.Fatalf("consume after failed attempt = (%v, %v), want Expired/nil", out, err)
	}

	// Re-issue overwrites and the correct code verifies exactly once.
	if err := store.Issue(ctx, phone, code, DefaultTTL); err != nil {
		t.Fatalf("re-Issue: %v", err)
	}
	if out, err := store.Consume(ctx, phone, code); err != nil || out != Verified {
		t.Fatalf("correct code = (%v, %v), want Verified/nil", out, err)
	}
	if out, err := store.Consume(ctx, phone, code); err != nil || out != Expired {
		t.Fatalf("replay = (%v, %v), want Expired/nil", out, err)
	}
}

func TestStoreTTL(t *testing.T) {
	mr := miniredis.RunT(t)
	store := NewStore(redis.NewClient(&redis.Options{Addr: mr.Addr()}), testSecret)
	ctx := context.Background()
	phone := "+5215511111111"

	if err := store.Issue(ctx, phone, "654321", DefaultTTL); err != nil {
		t.Fatalf("Issue: %v", err)
	}
	mr.FastForward(DefaultTTL + time.Minute)
	if out, err := store.Consume(ctx, phone, "654321"); err != nil || out != Expired {
		t.Fatalf("consume after TTL = (%v, %v), want Expired/nil", out, err)
	}
}

func TestSandboxSenderWritesToOutbox(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "outbox.jsonl")
	sender := NewSandboxSender(path)

	if err := sender.Send(context.Background(), "+5215511111111", "123456"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read outbox: %v", err)
	}
	var rec struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
		Kind  string `json:"kind"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(raw), &rec); err != nil {
		t.Fatalf("outbox line is not JSON: %v", err)
	}
	if rec.Phone != "+5215511111111" || rec.Code != "123456" || rec.Kind != "otp" {
		t.Fatalf("record = %+v, want phone/code/otp", rec)
	}
	if strings.Contains(path, "imalive/backend") && !strings.HasSuffix(path, filepath.Join("t", "outbox.jsonl")) {
		// the default path lives outside the repo; an explicit path is a test's choice
		t.Log("explicit test path in use")
	}
}
