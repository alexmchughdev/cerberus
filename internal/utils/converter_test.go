package utils

import "testing"

func TestExtractHTTPHost(t *testing.T) {
	var payload [32]byte
	copy(payload[:], []byte("Host: api.local\r\nUser-Agent: t"))
	host := ExtractHTTPHost(payload)
	if host != "api.local" {
		t.Fatalf("expected api.local, got %q", host)
	}
}

func TestInspectDNSDetailsQueryAndResponse(t *testing.T) {
	var query [32]byte
	// DNS header: QR=0, one question
	query[2] = 0x01
	query[3] = 0x00
	query[4] = 0x00
	query[5] = 0x01
	// QNAME "a.com"
	query[12] = 1
	query[13] = 'a'
	query[14] = 3
	query[15] = 'c'
	query[16] = 'o'
	query[17] = 'm'
	query[18] = 0
	// QTYPE A
	query[19] = 0
	query[20] = 1
	query[21] = 0
	query[22] = 1 // QCLASS IN

	domain, qtype, rcode, isResp := InspectDNSDetails(query)
	if domain != "a.com" {
		t.Fatalf("expected domain a.com, got %q", domain)
	}
	if qtype != "A" {
		t.Fatalf("expected qtype A, got %q", qtype)
	}
	if rcode != "NOERROR" {
		t.Fatalf("expected rcode NOERROR, got %q", rcode)
	}
	if isResp {
		t.Fatalf("expected query packet")
	}

	var response [32]byte
	response[2] = 0x81 // QR=1
	response[3] = 0x83 // RCODE=3 NXDOMAIN
	domain, qtype, rcode, isResp = InspectDNSDetails(response)
	if !isResp {
		t.Fatalf("expected response packet")
	}
	if rcode != "NXDOMAIN" {
		t.Fatalf("expected NXDOMAIN, got %q", rcode)
	}
	if domain != "" || qtype != "" {
		t.Fatalf("expected empty domain/qtype for minimal response payload")
	}
}

func TestTLSVersionFromPayload(t *testing.T) {
	var tls12 [32]byte
	tls12[0] = 0x16
	tls12[1] = 0x03
	tls12[2] = 0x03
	if got := TLSVersionFromPayload(tls12); got != "TLS 1.2" {
		t.Fatalf("expected TLS 1.2, got %q", got)
	}

	var tls13 [32]byte
	tls13[0] = 0x16
	tls13[1] = 0x03
	tls13[2] = 0x04
	if got := TLSVersionFromPayload(tls13); got != "TLS 1.3" {
		t.Fatalf("expected TLS 1.3, got %q", got)
	}

	var invalid [32]byte
	invalid[0] = 0x17
	if got := TLSVersionFromPayload(invalid); got != "" {
		t.Fatalf("expected empty version for invalid record, got %q", got)
	}
}
