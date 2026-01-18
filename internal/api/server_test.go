package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zrougamed/cerberus/internal/models"
	"github.com/zrougamed/cerberus/internal/monitor"
)

func TestTopMapOrdersAndLimits(t *testing.T) {
	input := map[string]int{
		"http": 8,
		"dns":  11,
		"tls":  4,
		"ssh":  9,
	}

	got := topMap(input, 2)
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	if got["dns"] != 11 {
		t.Fatalf("expected dns to be kept")
	}
	if got["ssh"] != 9 {
		t.Fatalf("expected ssh to be kept")
	}
	if _, exists := got["tls"]; exists {
		t.Fatalf("did not expect tls in top 2")
	}
}

func TestEscapeLabelValue(t *testing.T) {
	got := escapeLabelValue("Acme \"Corp\"\\line")
	want := "Acme \\\"Corp\\\"\\\\line"
	if got != want {
		t.Fatalf("unexpected escaped label: got %q want %q", got, want)
	}
}

func TestMetricsEndpointIncludesCoreCounters(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	mon, err := monitor.NewNetworkMonitor(10, dbPath)
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}
	defer func() {
		_ = mon.Close()
		_ = os.Remove(dbPath)
	}()

	mon.Stats.TotalPackets = 12
	mon.Stats.DnsPackets = 4
	now := time.Now()
	mon.Cache.Add("aa:bb:cc:dd:ee:ff", &models.DeviceInfo{
		MAC:            "aa:bb:cc:dd:ee:ff",
		Vendor:         `Acme "Corp"`,
		LastSeen:       now,
		FirstSeen:      now,
		DNSQueries:     3,
		HTTPRequests:   1,
		TLSConnections: 2,
		TCPConnections: 5,
		UDPConnections: 3,
		ICMPPackets:    0,
	})

	srv := NewServer(mon)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	srv.handleMetrics(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	body := rr.Body.String()
	mustContain := []string{
		`cerberus_packets_total{protocol="total"} 12`,
		`cerberus_packets_total{protocol="dns"} 4`,
		`cerberus_devices_total 1`,
		`cerberus_device_protocol_events_total{mac="aa:bb:cc:dd:ee:ff",vendor="Acme \"Corp\"",protocol="dns"} 3`,
	}
	for _, token := range mustContain {
		if !strings.Contains(body, token) {
			t.Fatalf("metrics body missing token %q", token)
		}
	}
}
