package webrtc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/pion/webrtc/v3"
)

// TestOfferCreatesSessions ensures each offer gets its own track.
func TestOfferCreatesSessions(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, "/webrtc")
	srv := httptest.NewServer(mux)
	defer srv.Close()

	sendOffer := func() {
		pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
		if err != nil {
			t.Fatalf("peer connection: %v", err)
		}
		defer pc.Close()
		if _, err := pc.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo); err != nil {
			t.Fatalf("add transceiver: %v", err)
		}
		offer, err := pc.CreateOffer(nil)
		if err != nil {
			t.Fatalf("create offer: %v", err)
		}
		gather := webrtc.GatheringCompletePromise(pc)
		if err := pc.SetLocalDescription(offer); err != nil {
			t.Fatalf("set local: %v", err)
		}
		<-gather
		buf := new(bytes.Buffer)
		_ = json.NewEncoder(buf).Encode(map[string]any{"sdp": pc.LocalDescription()})
		resp, err := http.Post(srv.URL+"/webrtc/offer", "application/json", buf)
		if err != nil {
			t.Fatalf("post offer: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
	}

	sendOffer()
	sendOffer()

	sessions := Sessions()
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	if sessions[0].Track == sessions[1].Track {
		t.Fatalf("tracks should be independent")
	}

	for _, s := range Sessions() {
		s.PC.Close()
	}
}

// TestIceServersFromEnv ensures ICE servers are parsed from environment.
func TestIceServersFromEnv(t *testing.T) {
	t.Setenv("ICE_SERVERS", "stun:stun.example.com, turn:turn.example.com")
	servers := iceServers()
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}
	if servers[0].URLs[0] != "stun:stun.example.com" {
		t.Fatalf("unexpected first server: %v", servers[0].URLs)
	}
}

// TestSessionConcurrency ensures map access is safe when sessions are created in parallel.
func TestSessionConcurrency(t *testing.T) {
	const n = 10
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sess, err := NewSession()
			if err != nil {
				t.Errorf("new session: %v", err)
				return
			}
			sess.PC.Close()
		}()
	}
	wg.Wait()
	time.Sleep(100 * time.Millisecond)
	if len(Sessions()) != 0 {
		t.Fatalf("expected 0 sessions after cleanup, got %d", len(Sessions()))
	}
}
