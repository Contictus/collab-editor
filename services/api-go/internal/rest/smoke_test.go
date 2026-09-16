package rest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestFullFlowSmoke drives the whole REST surface over real HTTP (routing,
// PathValue extraction, cookie jar): register → me → create → get → rename →
// share → audit → replay → logout. The F6 capstone.
func TestFullFlowSmoke(t *testing.T) {
	api := testAPI(t)
	api.Limiter = NewRateLimiter()
	mux := http.NewServeMux()
	api.Routes(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	suffix := time.Now().Format("150405.000000000")
	ownerEmail, guestEmail := "smoke-o-"+suffix+"@test.local", "smoke-g-"+suffix+"@test.local"

	call := func(method, path, body string, want int) []byte {
		t.Helper()
		var reader io.Reader
		if body != "" {
			reader = strings.NewReader(body)
		}
		req, err := http.NewRequest(method, srv.URL+path, reader)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != want {
			t.Fatalf("%s %s = %d %s, want %d", method, path, resp.StatusCode, raw, want)
		}
		return raw
	}

	raw := call("POST", "/api/auth/register", `{"email":"`+ownerEmail+`","password":"password123"}`, 201)
	var reg map[string]string
	_ = json.Unmarshal(raw, &reg)
	if reg["id"] == "" {
		t.Fatalf("register = %s", raw)
	}
	ownerID := reg["id"]
	var guestID string
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		docs, _ := api.Store.ListDocuments(ctx, ownerID)
		for _, d := range docs {
			_, _ = api.Store.DeleteDocument(ctx, d.ID, ownerID)
		}
		_ = api.Store.DeleteUser(ctx, ownerID)
		if guestID != "" {
			gdocs, _ := api.Store.ListDocuments(ctx, guestID)
			for _, d := range gdocs {
				_, _ = api.Store.DeleteDocument(ctx, d.ID, guestID)
			}
			_ = api.Store.DeleteUser(ctx, guestID)
		}
	})
	call("GET", "/api/auth/me", "", 200)

	raw = call("POST", "/api/documents", `{"title":"smoke doc"}`, 201)
	var created map[string]string
	_ = json.Unmarshal(raw, &created)
	id := created["id"]

	raw = call("GET", "/api/documents/"+id, "", 200)
	var detail documentDetailJSON
	_ = json.Unmarshal(raw, &detail)
	if detail.Title != "smoke doc" || detail.Text != "" || !detail.IsOwner {
		t.Fatalf("detail = %+v", detail)
	}

	call("PATCH", "/api/documents/"+id, `{"title":"smoke v2"}`, 200)
	raw = call("GET", "/api/documents", "", 200)
	var listed []documentJSON
	_ = json.Unmarshal(raw, &listed)
	if len(listed) != 1 || listed[0].Title != "smoke v2" {
		t.Fatalf("listed = %+v", listed)
	}

	// Second user, then share.
	jar2, _ := cookiejar.New(nil)
	guestClient := &http.Client{Jar: jar2}
	callGuest := func(method, path, body string, want int) []byte {
		t.Helper()
		var reader io.Reader
		if body != "" {
			reader = strings.NewReader(body)
		}
		req, _ := http.NewRequest(method, srv.URL+path, reader)
		resp, err := guestClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != want {
			t.Fatalf("guest %s %s = %d %s, want %d", method, path, resp.StatusCode, raw, want)
		}
		return raw
	}
	callGuest("POST", "/api/auth/register", `{"email":"`+guestEmail+`","password":"password123"}`, 201)
	{
		var greg map[string]string
		// Re-read the guest id from me for cleanup.
		req, _ := http.NewRequest("GET", srv.URL+"/api/auth/me", nil)
		resp, err := guestClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		_ = json.Unmarshal(raw, &greg)
		guestID = greg["id"]
	}
	callGuest("GET", "/api/documents/"+id, "", 404)
	call("POST", "/api/documents/"+id+"/share", `{"email":"`+guestEmail+`"}`, 201)
	callGuest("GET", "/api/documents/"+id, "", 200)

	// Audit + replay windows on the empty history.
	raw = call("GET", "/api/documents/"+id+"/audit", "", 200)
	var audit auditJSON
	_ = json.Unmarshal(raw, &audit)
	if audit.Counts.LiveUpdates != 0 || audit.BaseUpdateID != "0" || audit.LatestUpdateID != nil {
		t.Fatalf("audit = %+v", audit)
	}
	raw = call("GET", "/api/documents/"+id+"/replay?at=0", "", 200)
	var replay map[string]string
	_ = json.Unmarshal(raw, &replay)
	if replay["text"] != "" || replay["updateId"] != "0" {
		t.Fatalf("replay = %+v", replay)
	}

	// Logout ends the session.
	call("POST", "/api/auth/logout", "", 204)
	call("GET", "/api/auth/me", "", 401)
}
