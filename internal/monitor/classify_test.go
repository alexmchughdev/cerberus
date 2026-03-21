package monitor

import (
	"testing"
	"time"

	"github.com/zrougamed/cerberus/internal/models"
)

func TestClassifyDNSTraffic(t *testing.T) {
	nm := &NetworkMonitor{}

	var queryPayload [models.L7PayloadSize]byte
	queryPayload[2] = 0x01
	queryPayload[3] = 0x00
	if got := nm.classifyDNSTraffic(queryPayload); got != models.TrafficDNSQuery {
		t.Fatalf("expected DNS query, got %s", got)
	}

	var respPayload [models.L7PayloadSize]byte
	respPayload[2] = 0x81
	respPayload[3] = 0x80
	if got := nm.classifyDNSTraffic(respPayload); got != models.TrafficDNSResponse {
		t.Fatalf("expected DNS response, got %s", got)
	}
}

func TestClassifyHTTPTraffic(t *testing.T) {
	nm := &NetworkMonitor{}

	var getPayload [models.L7PayloadSize]byte
	copy(getPayload[:], []byte("GET /health HTTP/1.1"))
	if got := nm.classifyHTTPTraffic(getPayload); got != models.TrafficHTTPGET {
		t.Fatalf("expected HTTP GET, got %s", got)
	}

	var postPayload [models.L7PayloadSize]byte
	copy(postPayload[:], []byte("POST /events HTTP/1.1"))
	if got := nm.classifyHTTPTraffic(postPayload); got != models.TrafficHTTPPOST {
		t.Fatalf("expected HTTP POST, got %s", got)
	}

	var otherPayload [models.L7PayloadSize]byte
	copy(otherPayload[:], []byte("HEAD /index HTTP/1.1"))
	if got := nm.classifyHTTPTraffic(otherPayload); got != models.TrafficHTTPRequest {
		t.Fatalf("expected HTTP REQUEST fallback, got %s", got)
	}
}

func TestClassifyTLSTraffic(t *testing.T) {
	nm := &NetworkMonitor{}

	var clientHello [models.L7PayloadSize]byte
	clientHello[0] = 0x16
	clientHello[5] = 0x01
	if got := nm.classifyTLSTraffic(clientHello); got != models.TrafficTLSClientHello {
		t.Fatalf("expected TLS client hello, got %s", got)
	}

	var serverHello [models.L7PayloadSize]byte
	serverHello[0] = 0x16
	serverHello[5] = 0x02
	if got := nm.classifyTLSTraffic(serverHello); got != models.TrafficTLSServerHello {
		t.Fatalf("expected TLS server hello, got %s", got)
	}

	var handshake [models.L7PayloadSize]byte
	handshake[0] = 0x16
	handshake[5] = 0x03
	if got := nm.classifyTLSTraffic(handshake); got != models.TrafficTLSHandshake {
		t.Fatalf("expected TLS handshake fallback, got %s", got)
	}
}

func TestPickRecentDNSDomain(t *testing.T) {
	nm := &NetworkMonitor{}
	now := time.Now()
	device := &models.DeviceInfo{
		RecentDNSQueries: map[string]time.Time{
			"old.example": now.Add(-5 * time.Minute),
			"new.example": now.Add(-30 * time.Second),
		},
	}

	got := nm.pickRecentDNSDomain(device, now, 2*time.Minute)
	if got != "new.example" {
		t.Fatalf("expected new.example, got %q", got)
	}
	if _, exists := device.RecentDNSQueries["old.example"]; exists {
		t.Fatalf("expected old.example to be evicted")
	}
}

func TestIdentifyEncryptedDNS(t *testing.T) {
	nm := &NetworkMonitor{}

	if got := nm.identifyEncryptedDNS(models.EVENT_TYPE_TCP, 50000, 853, "192.168.1.5", "1.1.1.1"); got != "dot" {
		t.Fatalf("expected dot, got %q", got)
	}
	if got := nm.identifyEncryptedDNS(models.EVENT_TYPE_TLS, 50001, 443, "192.168.1.5", "1.1.1.1"); got != "doh" {
		t.Fatalf("expected doh, got %q", got)
	}
	if got := nm.identifyEncryptedDNS(models.EVENT_TYPE_TLS, 50001, 443, "192.168.1.5", "142.250.1.1"); got != "" {
		t.Fatalf("expected empty for non-resolver HTTPS destination, got %q", got)
	}
}

func TestIsGeoIPEligible(t *testing.T) {
	cases := []struct {
		ip   string
		want bool
	}{
		{ip: "8.8.8.8", want: true},
		{ip: "1.1.1.1", want: true},
		{ip: "192.168.1.1", want: false},
		{ip: "10.0.0.1", want: false},
		{ip: "127.0.0.1", want: false},
		{ip: "invalid-ip", want: false},
	}
	for _, tc := range cases {
		if got := isGeoIPEligible(tc.ip); got != tc.want {
			t.Fatalf("isGeoIPEligible(%q) = %v, want %v", tc.ip, got, tc.want)
		}
	}
}
