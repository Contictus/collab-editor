package sync

import (
	"testing"
	"time"

	"github.com/Contictus/collab-editor/services/api-go/internal/crdt"
	"net/http/httptest"
)

func TestCloseConnectionsDrains(t *testing.T) {
	az := stubAuthorizer{allow: map[string]bool{"room9\x00u1": true}}
	reg := NewRegistry(nil)
	ws := NewServer(testSecret, reg, az, 0)
	srv := httptest.NewServer(ws)
	t.Cleanup(srv.Close)

	conn, _ := dial(t, wsURL(srv, "/room9"), userCookie(t, "u1", "u@x.co"))
	defer conn.Close()
	if typ, _ := readFrame(t, conn); typ != crdt.MsgSync {
		t.Fatal("no step1")
	}

	ws.CloseConnections()
	deadline := time.Now().Add(5 * time.Second)
	for {
		rooms, conns := reg.Stats()
		if rooms == 0 && conns == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("drain = rooms %d conns %d", rooms, conns)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
