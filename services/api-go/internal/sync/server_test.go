package sync

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Contictus/collab-editor/services/api-go/internal/auth"
	"github.com/Contictus/collab-editor/services/api-go/internal/crdt"
	yawareness "github.com/reearth/ygo/awareness"
)

const testSecret = "server-test-secret"

func testServer(t *testing.T, az Authorizer) (*httptest.Server, *Registry) {
	t.Helper()
	reg := NewRegistry(nil)
	srv := httptest.NewServer(NewServer(testSecret, reg, az, 0))
	t.Cleanup(srv.Close)
	return srv, reg
}

func wsURL(srv *httptest.Server, path string) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http") + path
}

func userCookie(t *testing.T, id, email string) *http.Cookie {
	t.Helper()
	tok, err := auth.SignSession(auth.SessionUser{ID: id, Email: email}, testSecret)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: auth.SessionCookie, Value: tok}
}

func dial(t *testing.T, url string, cookie *http.Cookie) (*websocket.Conn, *http.Response) {
	t.Helper()
	h := http.Header{}
	if cookie != nil {
		h.Set("Cookie", cookie.String())
	}
	conn, resp, err := websocket.DefaultDialer.Dial(url, h)
	if err != nil && conn == nil {
		return nil, resp
	}
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	return conn, resp
}

func readFrame(t *testing.T, conn *websocket.Conn) (uint64, []byte) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	msgType, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if msgType != websocket.BinaryMessage {
		t.Fatalf("msg type = %d", msgType)
	}
	typ, payload, err := crdt.ParseFrame(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return typ, payload
}

func TestServerRejectsUnauthenticated(t *testing.T) {
	srv, _ := testServer(t, stubAuthorizer{allow: map[string]bool{}})
	if _, resp := dial(t, wsURL(srv, "/room1"), nil); resp == nil || resp.StatusCode != 401 {
		t.Fatalf("status = %+v", resp)
	}
	bad := &http.Cookie{Name: auth.SessionCookie, Value: "bogus"}
	if _, resp := dial(t, wsURL(srv, "/room1"), bad); resp == nil || resp.StatusCode != 401 {
		t.Fatalf("bogus status = %+v", resp)
	}
}

func TestServerRejectsForbidden(t *testing.T) {
	srv, _ := testServer(t, stubAuthorizer{allow: map[string]bool{}})
	cookie := userCookie(t, "u1", "u@x.co")
	if _, resp := dial(t, wsURL(srv, "/shared-no"), cookie); resp == nil || resp.StatusCode != 403 {
		t.Fatalf("status = %+v", resp)
	}
}

func TestServerCollabFlow(t *testing.T) {
	az := stubAuthorizer{allow: map[string]bool{"room1\x00u1": true, "room1\x00u2": true}}
	srv, reg := testServer(t, az)

	connA, _ := dial(t, wsURL(srv, "/room1"), userCookie(t, "u1", "a@x.co"))
	defer connA.Close()
	connB, _ := dial(t, wsURL(srv, "/room1"), userCookie(t, "u2", "b@x.co"))
	defer connB.Close()

	// Both open with a sync step1.
	if typ, _ := readFrame(t, connA); typ != crdt.MsgSync {
		t.Fatalf("A opener = %d", typ)
	}
	if typ, _ := readFrame(t, connB); typ != crdt.MsgSync {
		t.Fatalf("B opener = %d", typ)
	}

	// Awareness relay: A publishes presence, B receives it.
	aw := yawareness.New(424242)
	aw.SetLocalState(map[string]any{"user": map[string]any{"name": "A"}})
	defer aw.Destroy()
	if err := connA.WriteMessage(websocket.BinaryMessage,
		crdt.EncodeAwarenessFrame(aw.EncodeUpdate(nil))); err != nil {
		t.Fatalf("aw write: %v", err)
	}
	typ, payload := readFrame(t, connB)
	if typ != crdt.MsgAwareness {
		t.Fatalf("B awareness = %d", typ)
	}
	probe := crdt.NewPresence()
	defer probe.Destroy()
	if err := probe.Apply(payload, nil); err != nil {
		t.Fatalf("probe apply: %v", err)
	}
	st, ok := probe.States()[424242]
	if !ok || st.State == nil {
		t.Fatal("presence not relayed")
	}

	// Doc relay: A inserts text, server applies + broadcasts, B converges.
	docA := crdt.New()
	defer docA.Destroy()
	var update []byte
	unsub := docA.OnUpdate(func(u []byte, _ any) { update = u })
	docA.InsertText(0, "hi from A")
	unsub()
	if err := connA.WriteMessage(websocket.BinaryMessage, crdt.EncodeUpdateFrame(update)); err != nil {
		t.Fatalf("update write: %v", err)
	}
	typ, payload = readFrame(t, connB)
	if typ != crdt.MsgSync {
		t.Fatalf("B update = %d", typ)
	}
	docB := crdt.New()
	defer docB.Destroy()
	if _, err := crdt.HandleSyncMessage(docB, payload, nil); err != nil {
		t.Fatalf("B apply: %v", err)
	}
	if got := docB.Text(); got != "hi from A" {
		t.Fatalf("B = %q", got)
	}

	// Server replica converged too (authoritative, invariant #4).
	room, err := reg.Get(t.Context(), "room1")
	if err != nil {
		t.Fatal(err)
	}
	if got := room.Document().Text(); got != "hi from A" {
		t.Fatalf("server = %q", got)
	}
}
