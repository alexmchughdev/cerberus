package api

import "testing"

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
