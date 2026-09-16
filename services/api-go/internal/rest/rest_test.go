package rest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteErrShape(t *testing.T) {
	w := httptest.NewRecorder()
	writeErr(w, 401, "unauthorized")
	if w.Code != 401 {
		t.Fatalf("code = %d", w.Code)
	}
	if got := strings.TrimSpace(w.Body.String()); got != `{"error":"unauthorized"}` {
		t.Fatalf("body = %s", got)
	}
	if ct := w.Header().Get("content-type"); ct != "application/json" {
		t.Fatalf("ct = %s", ct)
	}
}

func TestDecodeBody(t *testing.T) {
	type form struct {
		Email string `json:"email"`
	}
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"a@b.co"}`))
	v, ok := decodeBody[form](r)
	if !ok || v.Email != "a@b.co" {
		t.Fatalf("= %+v %v", v, ok)
	}
	bad := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{oops`))
	if _, ok := decodeBody[form](bad); ok {
		t.Fatal("malformed accepted")
	}
	empty := httptest.NewRequest(http.MethodPost, "/", nil)
	if _, ok := decodeBody[form](empty); ok {
		t.Fatal("empty accepted")
	}
}
