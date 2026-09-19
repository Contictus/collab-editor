package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/Contictus/collab-editor/services/api-go/internal/crdt"
	gosync "github.com/Contictus/collab-editor/services/api-go/internal/sync"
)

func postRestore(t *testing.T, api *API, id, at string, session *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/documents/"+id+"/restore",
		strings.NewReader(`{"at":"`+at+`"}`))
	if session != nil {
		r.AddCookie(session)
	}
	api.handleRestore(w, r, id)
	return w
}

// Restore replays a past point into the live room: anon 401, bad body 400,
// unknown point 404, happy path 200 with the room text replaced.
func TestRestoreEndpoint(t *testing.T) {
	api := testAPI(t)
	api.Rooms = gosync.NewRegistry(nil)
	_, owner := authedUser(t, api, "rsowner")
	id := createDoc(t, api, owner, "restore me")

	if w := postRestore(t, api, id, "1", nil); w.Code != 401 {
		t.Fatalf("anon = %d", w.Code)
	}
	if w := postRestore(t, api, id, "bogus", owner); w.Code != 400 {
		t.Fatalf("bad at = %d", w.Code)
	}
	if w := postRestore(t, api, id, "7", owner); w.Code != 404 {
		t.Fatalf("beyond empty = %d", w.Code)
	}

	// Two sequential updates ("Hello", "Hello world").
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

	w := postRestore(t, api, id, at1, owner)
	var body map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if w.Code != 200 || body["text"] != "Hello" || body["updateId"] != at1 {
		t.Fatalf("restore = %d %s", w.Code, w.Body.String())
	}
	room, err := api.Rooms.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got := room.Document().Text(); got != "Hello" {
		t.Fatalf("room text = %q", got)
	}
}
