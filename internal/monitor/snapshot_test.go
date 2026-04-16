package monitor

import (
	"testing"

	"github.com/zrougamed/cerberus/internal/models"
)

func TestCloneDeviceInfoDoesNotAliasMaps(t *testing.T) {
	t.Parallel()
	orig := &models.DeviceInfo{
		MAC:      "aa:bb:cc:dd:ee:ff",
		Services: map[string]int{"http": 1},
		TrafficTypeCounts: map[models.TrafficType]int{
			models.TrafficTCPSYN: 2,
		},
		Targets: []string{"192.168.0.1"},
	}
	cl := cloneDeviceInfo(orig)
	cl.Services["http"] = 999
	cl.TrafficTypeCounts[models.TrafficTCPSYN] = 0
	cl.Targets[0] = "10.0.0.1"

	if orig.Services["http"] != 1 {
		t.Fatalf("Services map aliased: got %d", orig.Services["http"])
	}
	if orig.TrafficTypeCounts[models.TrafficTCPSYN] != 2 {
		t.Fatalf("TrafficTypeCounts aliased: got %d", orig.TrafficTypeCounts[models.TrafficTCPSYN])
	}
	if orig.Targets[0] != "192.168.0.1" {
		t.Fatalf("Targets slice aliased: got %q", orig.Targets[0])
	}
}
