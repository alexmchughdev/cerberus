package databases

import (
	"path/filepath"
	"testing"
)

func TestParseMAC48(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want [6]byte
		ok   bool
	}{
		{"aa:bb:cc:dd:ee:ff", [6]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}, true},
		{"AA-BB-CC-DD-EE-FF", [6]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}, true},
		{"aabbccddeeff", [6]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}, true},
		{"01:00:5e:00:00:fb", [6]byte{0x01, 0x00, 0x5E, 0x00, 0x00, 0xFB}, true},
		{"bad", [6]byte{}, false},
		{"aa:bb:cc", [6]byte{}, false},
	}
	for _, tc := range cases {
		got, ok := ParseMAC48(tc.in)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("ParseMAC48(%q) = (%v,%v) want (%v,%v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestOUIKeyCandidatesOrder(t *testing.T) {
	t.Parallel()
	mac := [6]byte{0x28, 0x6A, 0xBA, 0x01, 0x02, 0x03}
	keys := OUIKeyCandidates(mac)
	if len(keys) != 3 || keys[0] != "28:6A:BA:01:02" || keys[1] != "28:6A:BA:01" || keys[2] != "28:6A:BA" {
		t.Fatalf("unexpected keys: %#v", keys)
	}
}

func TestOUIDatabaseLookupSpecialBits(t *testing.T) {
	t.Setenv("CERBERUS_DATA_DIR", filepath.Join(t.TempDir(), "d"))
	db, err := NewOUIDatabase(false)
	if err != nil {
		t.Fatal(err)
	}
	if got := db.Lookup("33:33:00:00:00:01"); got != "Multicast" {
		t.Fatalf("multicast: got %q", got)
	}
	if got := db.Lookup("02:00:00:00:00:01"); got != "Locally administered address" {
		t.Fatalf("local: got %q", got)
	}
	if got := db.Lookup("B8:27:EB:00:00:01"); got != "Raspberry Pi Foundation" {
		t.Fatalf("fallback OUI: got %q", got)
	}
}
