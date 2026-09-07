package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
)

func TestHTTPReadinessAndObservedHealth(t *testing.T) {
	var status atomic.Int32
	status.Store(200)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(int(status.Load())) }))
	defer server.Close()
	p := config.Probe{Type: "http", URL: server.URL, Timeout: "20ms", Interval: "1ms"}
	m := config.Manifest{Components: map[string]config.Component{"api": {Readiness: []config.Probe{p}}}}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	l := domain.Lease{Components: []domain.Component{{Name: "api"}}, Manifest: data}
	s := &Service{}
	if err := s.probeReady(context.Background(), &l); err != nil {
		t.Fatal(err)
	}
	status.Store(503)
	if err := s.probeHealth(context.Background(), l); err == nil {
		t.Fatal("failed HTTP health still reported healthy")
	}
	start := time.Now()
	if err := s.probeReady(context.Background(), &l); err == nil {
		t.Fatal("failed readiness accepted")
	}
	if time.Since(start) > time.Second {
		t.Fatal("probe timeout not bounded")
	}
}
