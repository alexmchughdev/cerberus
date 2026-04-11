package monitor

import (
	"testing"
	"time"

	"github.com/zrougamed/cerberus/internal/models"
)

func TestAnomalyDetectorEmitsAnomalyAfterBaseline(t *testing.T) {
	ad := newAnomalyDetector()
	ad.window = 50 * time.Millisecond
	ad.baselineNeeded = 3
	start := time.Now()

	// Build baseline windows
	for w := 0; w < 3; w++ {
		for i := 0; i < 5; i++ {
			ad.observe(start.Add(time.Duration(w)*ad.window+time.Duration(i)*time.Millisecond), &models.NetworkEvent{EventType: models.EVENT_TYPE_TCP, DstPort: 443}, "aa:bb")
		}
		ad.observe(start.Add(time.Duration(w+1)*ad.window), &models.NetworkEvent{EventType: models.EVENT_TYPE_TCP, DstPort: 443}, "aa:bb")
	}

	// Inject spiky anomalous window
	for i := 0; i < 100; i++ {
		ad.observe(start.Add(4*ad.window+time.Duration(i)*time.Microsecond), &models.NetworkEvent{EventType: models.EVENT_TYPE_DNS, DstPort: 60000}, "cc:dd")
	}
	ad.observe(start.Add(5*ad.window), &models.NetworkEvent{EventType: models.EVENT_TYPE_DNS, DstPort: 60001}, "cc:dd")

	s := ad.status()
	if s.Status != "active" {
		t.Fatalf("expected active status, got %q", s.Status)
	}
	if len(s.RecentAlerts) == 0 {
		t.Fatalf("expected anomaly alerts to be produced")
	}
}
