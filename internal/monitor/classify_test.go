package monitor

import (
	"testing"

	"github.com/zrougamed/cerberus/internal/models"
)

func TestClassifyDNSTraffic(t *testing.T) {
	nm := &NetworkMonitor{}

	var queryPayload [32]byte
	queryPayload[2] = 0x01
	queryPayload[3] = 0x00
	if got := nm.classifyDNSTraffic(queryPayload); got != models.TrafficDNSQuery {
		t.Fatalf("expected DNS query, got %s", got)
	}

	var respPayload [32]byte
	respPayload[2] = 0x81
	respPayload[3] = 0x80
	if got := nm.classifyDNSTraffic(respPayload); got != models.TrafficDNSResponse {
		t.Fatalf("expected DNS response, got %s", got)
	}
}

func TestClassifyHTTPTraffic(t *testing.T) {
	nm := &NetworkMonitor{}

	var getPayload [32]byte
	copy(getPayload[:], []byte("GET /health HTTP/1.1"))
	if got := nm.classifyHTTPTraffic(getPayload); got != models.TrafficHTTPGET {
		t.Fatalf("expected HTTP GET, got %s", got)
	}

	var postPayload [32]byte
	copy(postPayload[:], []byte("POST /events HTTP/1.1"))
	if got := nm.classifyHTTPTraffic(postPayload); got != models.TrafficHTTPPOST {
		t.Fatalf("expected HTTP POST, got %s", got)
	}

	var otherPayload [32]byte
	copy(otherPayload[:], []byte("HEAD /index HTTP/1.1"))
	if got := nm.classifyHTTPTraffic(otherPayload); got != models.TrafficHTTPRequest {
		t.Fatalf("expected HTTP REQUEST fallback, got %s", got)
	}
}

func TestClassifyTLSTraffic(t *testing.T) {
	nm := &NetworkMonitor{}

	var clientHello [32]byte
	clientHello[0] = 0x16
	clientHello[5] = 0x01
	if got := nm.classifyTLSTraffic(clientHello); got != models.TrafficTLSClientHello {
		t.Fatalf("expected TLS client hello, got %s", got)
	}

	var serverHello [32]byte
	serverHello[0] = 0x16
	serverHello[5] = 0x02
	if got := nm.classifyTLSTraffic(serverHello); got != models.TrafficTLSServerHello {
		t.Fatalf("expected TLS server hello, got %s", got)
	}

	var handshake [32]byte
	handshake[0] = 0x16
	handshake[5] = 0x03
	if got := nm.classifyTLSTraffic(handshake); got != models.TrafficTLSHandshake {
		t.Fatalf("expected TLS handshake fallback, got %s", got)
	}
}
