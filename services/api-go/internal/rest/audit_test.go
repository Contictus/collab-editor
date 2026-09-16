package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Contictus/collab-editor/services/api-go/internal/crdt"
)

func getAudit(t *testing.T, api *API, id string, session *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/documents/"+id+"/audit", nil)
	if session != nil {
		r.AddCookie(session)
	}
	api.handleAudit(w, r, id)
	return w
}

func TestAudit(t *testing.T) {
	api := testAPI(t)
	_, owner := authedUser(t, api, "auowner")
	_, guest := authedUser(t, api, "auguess")
	id := createDoc(t, api, owner, "audited")

	if w := getAudit(t, api, id, nil); w.Code != 401 {
		t.Fatalf("anon = %d", w.Code)
	}
	if w := getAudit(t, api, id, guest); w.Code != 404 {
		t.Fatalf("guest = %d", w.Code)
	}

	// Two live updates, no snapshots yet.
	tmp := crdt.New()
	var update []byte
	unsub := tmp.OnUpdate(func(u []byte, _ any) { update = u })
	tmp.InsertText(0, "audit me")
	unsub()
	tmp.Destroy()
	ctx := context.Background()
	if err := api.Store.AppendUpdate(ctx, id, update, 0); err != nil {
		t.Fatal(err)
	}
	if err := api.Store.AppendUpdate(ctx, id, update, 1); err != nil {
		t.Fatal(err)
	}

	w := getAudit(t, api, id, owner)
	if w.Code != 200 {
		t.Fatalf("audit = %d %s", w.Code, w.Body.String())
	}
	var out auditJSON
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Counts.LiveUpdates != 2 || out.Counts.Snapshots != 0 {
		t.Fatalf("counts = %+v", out.Counts)
	}
	if out.BaseUpdateID != "0" || out.LatestUpdateID == nil {
		t.Fatalf("window = %q %+v", out.BaseUpdateID, out.LatestUpdateID)
	}
	if len(out.Updates) != 2 || out.Updates[0].Clock != 0 || out.Updates[1].Clock != 1 {
		t.Fatalf("updates = %+v", out.Updates)
	}

	// Compact: 1 snapshot, empty tail, base at the boundary.
	full, err := crdt.LoadText(nil, [][]byte{update})
	if err != nil || full != "audit me" {
		t.Fatal(err)
	}
	stateDoc := crdt.New()
	stateDoc.InsertText(0, "audit me")
	if _, err := api.Store.Compact(ctx, id, stateDoc.FullState()); err != nil {
		t.Fatal(err)
	}
	stateDoc.Destroy()
	w2 := getAudit(t, api, id, owner)
	var out2 auditJSON
	_ = json.Unmarshal(w2.Body.Bytes(), &out2)
	if out2.Counts.Snapshots != 1 || out2.Counts.LiveUpdates != 0 {
		t.Fatalf("counts2 = %+v", out2.Counts)
	}
	if out2.LatestUpdateID != nil || out2.BaseUpdateID == "0" {
		t.Fatalf("window2 = %q %+v", out2.BaseUpdateID, out2.LatestUpdateID)
	}
}
