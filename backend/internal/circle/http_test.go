package circle_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/tiagofur/imalive/backend/internal/auth"
	authstore "github.com/tiagofur/imalive/backend/internal/auth/store"
	"github.com/tiagofur/imalive/backend/internal/circle"
	"github.com/tiagofur/imalive/backend/internal/circle/store"
	"github.com/tiagofur/imalive/backend/internal/platform/db/dbtest"
	"github.com/tiagofur/imalive/backend/internal/platform/msg"
	"github.com/tiagofur/imalive/backend/internal/platform/otp"
)

type codeRecorder struct{ codes map[string]string }

func newCodeRecorder() *codeRecorder { return &codeRecorder{codes: map[string]string{}} }

func (c *codeRecorder) Send(_ context.Context, phone, code string) error {
	c.codes[phone] = code
	return nil
}

func (c *codeRecorder) last(phone string) string { return c.codes[phone] }

// api mounts the real stack (migrated Postgres + miniredis + auth + circle)
// with a sandbox notifier pointed at a test outbox file.
type api struct {
	server   *httptest.Server
	svc      *auth.Service
	sender   *codeRecorder
	notifier *msg.SandboxNotifier
}

func newAPI(t *testing.T, outboxPath string) *api {
	t.Helper()
	p := dbtest.New(t)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	sender := newCodeRecorder()
	notifier := msg.NewSandboxNotifier(outboxPath)

	svc := auth.NewService(otp.NewStore(rdb, "test-secret"), sender, rdb,
		authstore.New(p.Pool), "test-secret")
	mux := http.NewServeMux()
	auth.NewHandlers(svc, auth.NewRateLimiter(rdb)).Register(mux)
	circle.NewHandlers(store.New(p.Pool), notifier).Register(mux, auth.NewMiddleware(svc))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &api{server: srv, svc: svc, sender: sender, notifier: notifier}
}

func (a *api) signUp(t *testing.T, phone string) string {
	t.Helper()
	if err := a.svc.Request(context.Background(), phone); err != nil {
		t.Fatalf("Request: %v", err)
	}
	pair, err := a.svc.Verify(context.Background(), phone, a.sender.last(phone))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	return pair.AccessToken
}

func callAPI(t *testing.T, method, url, token string, payload any) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), method, url, bodyFor(t, payload))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

func bodyFor(t *testing.T, payload any) *bytes.Reader {
	t.Helper()
	if payload == nil {
		return bytes.NewReader(nil)
	}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return bytes.NewReader(b)
}

func decodeBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return m
}

func outboxTexts(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var texts []string
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		var rec struct {
			Text string `json:"text"`
		}
		if json.Unmarshal([]byte(line), &rec) == nil && rec.Text != "" {
			texts = append(texts, rec.Text)
		}
	}
	return texts
}

func TestContactsFlow(t *testing.T) {
	out, err := os.CreateTemp(t.TempDir(), "outbox-*.jsonl")
	if err != nil {
		t.Fatalf("temp outbox: %v", err)
	}
	_ = out.Close()
	api := newAPI(t, out.Name())
	phone := "+5215566770001"
	token := api.signUp(t, phone)

	t.Run("create returns 201 with the contract shape", func(t *testing.T) {
		resp := callAPI(t, http.MethodPost, api.server.URL+"/v1/contacts", token,
			map[string]string{"name": "María", "phone": "+5215588880001"})
		if resp.StatusCode != http.StatusCreated {
			body := decodeBody(t, resp)
			t.Fatalf("status = %d, body = %v; want 201", resp.StatusCode, body)
		}
		body := decodeBody(t, resp)
		if body["name"] != "María" || body["phone"] != "+5215588880001" || body["id"] == nil {
			t.Fatalf("created = %v, want id/name/phone", body)
		}
		if _, ok := body["createdAt"]; !ok {
			t.Fatalf("created = %v, want createdAt", body)
		}
	})

	t.Run("the welcome message goes to the sandbox outbox with the brief text", func(t *testing.T) {
		// async delivery: poll briefly
		var texts []string
		for range 20 {
			texts = outboxTexts(out.Name())
			if len(texts) > 0 {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		if len(texts) == 0 {
			t.Fatal("no welcome message in the sandbox outbox")
		}
		want := "Usuario te ha añadido como persona de confianza en Tranqui"
		if !strings.Contains(texts[0], want) {
			t.Fatalf("welcome = %q, want it to contain %q", texts[0], want)
		}
		if !strings.Contains(texts[0], "No tienes que hacer nada") {
			t.Fatalf("welcome = %q, want the brief's anti-CTA line", texts[0])
		}
		if !strings.Contains(texts[0], "no es un servicio de emergencias") {
			t.Fatalf("welcome = %q, want the honest disclaimer", texts[0])
		}
	})

	t.Run("list returns the ordered circle", func(t *testing.T) {
		second := callAPI(t, http.MethodPost, api.server.URL+"/v1/contacts", token,
			map[string]string{"name": "Luis", "phone": "+5215588880002"})
		if second.StatusCode != http.StatusCreated {
			t.Fatalf("second create = %d, want 201", second.StatusCode)
		}
		_ = second.Body.Close()
		resp := callAPI(t, http.MethodGet, api.server.URL+"/v1/contacts", token, nil)
		body := decodeBody(t, resp)
		contacts, ok := body["contacts"].([]any)
		if !ok || len(contacts) != 2 {
			t.Fatalf("list = %v, want 2 contacts", body)
		}
		if contacts[0].(map[string]any)["name"] != "María" {
			t.Fatalf("order = %v, want insertion order (position)", body)
		}
	})

	t.Run("duplicate phone in the circle is rejected", func(t *testing.T) {
		resp := callAPI(t, http.MethodPost, api.server.URL+"/v1/contacts", token,
			map[string]string{"name": "María Dup", "phone": "+5215588880001"})
		body := decodeBody(t, resp)
		if resp.StatusCode != http.StatusBadRequest || body["code"] != "VALIDATION_FAILED" {
			t.Fatalf("duplicate = %d %v; want 400 VALIDATION_FAILED", resp.StatusCode, body["code"])
		}
	})

	t.Run("the sixth contact yields 422 CONTACT_LIMIT_REACHED", func(t *testing.T) {
		for i := 3; i <= 5; i++ {
			resp := callAPI(t, http.MethodPost, api.server.URL+"/v1/contacts", token,
				map[string]string{"name": fmt.Sprintf("C%d", i), "phone": fmt.Sprintf("+52155888800%02d", i)})
			if resp.StatusCode != http.StatusCreated {
				t.Fatalf("create #%d = %d, want 201", i, resp.StatusCode)
			}
			_ = resp.Body.Close()
		}
		resp := callAPI(t, http.MethodPost, api.server.URL+"/v1/contacts", token,
			map[string]string{"name": "Sexto", "phone": "+5215588880006"})
		body := decodeBody(t, resp)
		if resp.StatusCode != http.StatusUnprocessableEntity || body["code"] != "CONTACT_LIMIT_REACHED" {
			t.Fatalf("sixth = %d %v; want 422 CONTACT_LIMIT_REACHED", resp.StatusCode, body["code"])
		}
	})

	t.Run("delete removes and unknown ids yield 404", func(t *testing.T) {
		list := callAPI(t, http.MethodGet, api.server.URL+"/v1/contacts", token, nil)
		contacts := decodeBody(t, list)["contacts"].([]any)
		firstID := contacts[0].(map[string]any)["id"].(string)

		resp := callAPI(t, http.MethodDelete, api.server.URL+"/v1/contacts/"+firstID, token, nil)
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("delete = %d, want 204", resp.StatusCode)
		}
		again := callAPI(t, http.MethodDelete, api.server.URL+"/v1/contacts/"+firstID, token, nil)
		body := decodeBody(t, again)
		if again.StatusCode != http.StatusNotFound || body["code"] != "CONTACT_NOT_FOUND" {
			t.Fatalf("re-delete = %d %v; want 404 CONTACT_NOT_FOUND", again.StatusCode, body["code"])
		}
	})

	t.Run("users are isolated: another user sees an empty circle", func(t *testing.T) {
		otherToken := api.signUp(t, "+5215566770002")
		resp := callAPI(t, http.MethodGet, api.server.URL+"/v1/contacts", otherToken, nil)
		body := decodeBody(t, resp)
		if contacts := body["contacts"].([]any); len(contacts) != 0 {
			t.Fatalf("other user's circle = %v, want empty", body)
		}
	})

	t.Run("no token yields 401", func(t *testing.T) {
		resp := callAPI(t, http.MethodGet, api.server.URL+"/v1/contacts", "", nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", resp.StatusCode)
		}
	})
}
