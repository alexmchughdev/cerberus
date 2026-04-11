package databases

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/zrougamed/cerberus/internal/models"
)

func TestExpandIANAPortField(t *testing.T) {
	t.Parallel()
	p, ok := expandIANAPortField("5222-5224")
	if !ok || len(p) != 3 || p[0] != 5222 || p[1] != 5223 || p[2] != 5224 {
		t.Fatalf("range: %#v ok=%v", p, ok)
	}
	wide, ok := expandIANAPortField("1000-3000")
	if !ok || len(wide) != 1 || wide[0] != 1000 {
		t.Fatalf("wide span should collapse to low port: %#v ok=%v", wide, ok)
	}
}

func TestParseIANACSVQuotesAndCommas(t *testing.T) {
	t.Parallel()
	db := &ServiceDatabase{
		services:    make(map[uint16]*models.ServiceInfo),
		tcpServices: make(map[uint16]*models.ServiceInfo),
		udpServices: make(map[uint16]*models.ServiceInfo),
		threatPorts: make(map[uint16]ThreatInfo),
	}
	csv := "Service Name,Port Number,Transport Protocol,Description,Assignee,Contact,Registration Date,Modification Date,Reference,Service Code,Unauthorized Use Reported,Assignment Notes\n" +
		`foo,80,tcp,"Hypertext Transfer Protocol, with commas",,,,,,,,` + "\n" +
		`,443,tcp,"HTTPS, secure",,,,,,,,` + "\n"

	n := db.parseIANACSV(csv)
	if n != 2 {
		t.Fatalf("expected 2 rows, got %d", n)
	}
	h := db.Lookup(80, "TCP")
	if h.Service != "FOO" || !strings.Contains(h.Description, "commas") {
		t.Fatalf("tcp 80: %+v", h)
	}
	h2 := db.Lookup(443, "TCP")
	if h2.Service != "TCP-443" {
		t.Fatalf("tcp 443 service: %+v", h2)
	}
}

func TestLoadServiceDatabaseUsesDataDir(t *testing.T) {
	t.Setenv("CERBERUS_DATA_DIR", filepath.Join(t.TempDir(), "svc"))
	m := LoadServiceDatabase()
	if len(m) == 0 {
		t.Fatal("expected fallback services")
	}
	if _, ok := m[53]; !ok {
		t.Fatal("expected DNS/UDP 53 in fallback")
	}
}
