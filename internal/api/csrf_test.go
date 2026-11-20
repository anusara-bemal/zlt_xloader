package api

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/anusara-bemal/zlt_xloader/internal/profile"
)

func TestLoginFlow(t *testing.T) {
	const challenge = "0.123456789"
	const token = "session-token-123"

	var gotForm url.Values
	var gotHeader string
	var calls []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/reqproc/proc_get":
			cmd := r.URL.Query().Get("cmd")
			calls = append(calls, cmd)
			switch cmd {
			case "get_token":
				fmt.Fprintf(w, `{"token":%q}`, token)
			case "get_random_login":
				fmt.Fprintf(w, `{"random_login":%q}`, challenge)
			default:
				fmt.Fprint(w, `{"result":"failure"}`)
			}
		case "/reqproc/proc_post":
			calls = append(calls, "LOGIN")
			if err := r.ParseForm(); err != nil {
				t.Errorf("ParseForm: %v", err)
			}
			gotForm = r.PostForm
			gotHeader = r.Header.Get("CSRFToken")
			if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
				t.Errorf("Content-Type = %q", ct)
			}
			fmt.Fprint(w, `{"result":"0"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second, false, profile.Builtin().Endpoints)
	if err := c.Login("admin", "secret"); err != nil {
		t.Fatalf("Login: %v", err)
	}

	if gotForm.Get("goformId") != "LOGIN" {
		t.Errorf("goformId = %q", gotForm.Get("goformId"))
	}
	if gotForm.Get("isTest") != "false" {
		t.Errorf("isTest = %q", gotForm.Get("isTest"))
	}
	wantUser := base64.StdEncoding.EncodeToString([]byte("admin"))
	if got := gotForm.Get("username"); got != wantUser {
		t.Errorf("username = %q, want %q", got, wantUser)
	}
	sum := sha256.Sum256([]byte(challenge + "secret"))
	wantPass := base64.StdEncoding.EncodeToString([]byte(hex.EncodeToString(sum[:])))
	if got := gotForm.Get("password"); got != wantPass {
		t.Errorf("password = %q, want %q", got, wantPass)
	}
	if got := gotForm.Get("CSRFToken"); got != "" {
		t.Errorf("LOGIN form CSRFToken = %q, want empty", got)
	}
	if gotHeader != "" {
		t.Errorf("LOGIN header CSRFToken = %q, want empty", gotHeader)
	}
	if c.Token() != token {
		t.Errorf("client token = %q, want %q", c.Token(), token)
	}

	wantCalls := []string{"get_random_login", "LOGIN", "get_token"}
	if len(calls) != len(wantCalls) {
		t.Fatalf("call order = %v, want %v", calls, wantCalls)
	}
	for i := range wantCalls {
		if calls[i] != wantCalls[i] {
			t.Fatalf("call order = %v, want %v", calls, wantCalls)
		}
	}
}

func TestLoginRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("cmd") {
		case "get_token":
			fmt.Fprint(w, `{"token":"t"}`)
		case "get_random_login":
			fmt.Fprint(w, `{"random_login":"r"}`)
		default:
			fmt.Fprint(w, `{"result":"3"}`)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2*time.Second, false, profile.Builtin().Endpoints)
	err := c.Login("admin", "wrong")
	if err == nil || !strings.Contains(err.Error(), "bad password") {
		t.Fatalf("expected bad password error, got %v", err)
	}
}
