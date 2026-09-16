package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Contictus/collab-editor/services/api-go/internal/crdt"
)

func getReplay(t *testing.T, api *API, id, at string, session *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/documents/"+id+"/replay?at="+at, nil)
	if session != nil {
		r.AddCookie(session)
	}
	api.handleReplay(w, r, id)
	return w
}

func TestReplayEndpoint(t *testing.T) {
	api := testAPI(t)
	_, owner := authedUser(t, api, "rpowner")
	id := createDoc(t, api, owner, "replay me")

	if w := getReplay(t, api, id, "1", nil); w.Code != 401 {
		t.Fatalf("anon = %d", w.Code)
	}
	if w := getReplay(t, api, id, "bogus", owner); w.Code != 400 {
		t.Fatalf("bad at = %d", w.Code)
	}
	if w := getReplay(t, api, id, "", owner); w.Code != 400 {
		t.Fatalf("empty at = %d", w.Code)
	}
	if w := getReplay(t, api, id, "7", owner); w.Code != 404 {
		t.Fatalf("beyond empty = %d", w.Code)
	}

	// Two sequential updates ("Hello", " world").
	tmp := crdt.New()
	var updates [][]byte
	unsub := tmp.OnUpdate(func(u []byte, _ any) { updates = append(updates, u) })
	tmp.InsertText(0, "Hello")
	tmp.InsertText(5, " world")
	unsub()
	tmp.Destroy()
	ctx := context.Background()
	for i, u := range updates {
		if err := api.Store.AppendUpdate(ctx, id, u, i); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := api.Store.ReplayUpdates(ctx, id)
	if err != nil || len(rows) != 2 {
		t.Fatal(err)
	}
	at1 := strconv.FormatInt(rows[0].ID, 10)
	at2 := strconv.FormatInt(rows[1].ID, 10)

	w := getReplay(t, api, id, "0", owner)
	var r0 map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &r0)
	if w.Code != 200 || r0["text"] != "" || r0["updateId"] != "0" {
		t.Fatalf("at 0 = %d %s", w.Code, w.Body.String())
	}
	w1 := getReplay(t, api, id, at1, owner)
	var r1 map[string]string
	_ = json.Unmarshal(w1.Body.Bytes(), &r1)
	if w1.Code != 200 || r1["text"] != "Hello" || r1["updateId"] != at1 {
		t.Fatalf("at 1 = %d %s", w1.Code, w1.Body.String())
	}
	w2 := getReplay(t, api, id, at2, owner)
	var r2 map[string]string
	_ = json.Unmarshal(w2.Body.Bytes(), &r2)
	if w2.Code != 200 || r2["text"] != "Hello world" {
		t.Fatalf("at 2 = %d %s", w2.Code, w2.Body.String())
	}
	past := strconv.FormatInt(rows[1].ID+1, 10)
	if w := getReplay(t, api, id, past, owner); w.Code != 404 {
		t.Fatalf("past latest = %d", w.Code)
	}
}
