package monitor

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/zrougamed/cerberus/internal/models"
)

// Per-feature robust z is capped so near-constant baselines cannot explode the score.
const perFeatureZCap = 8.0

var anomalyFeatureMeta = []struct {
	Key   string
	Label string
}{
	{"packet_rate", "Events per second (all types)"},
	{"dns_rate", "DNS events per second"},
	{"http_rate", "HTTP events per second"},
	{"tls_rate", "TLS events per second"},
	{"tcp_syn_rate", "TCP SYN packets per second"},
	{"unique_device_count", "Distinct devices seen in window"},
	{"unusual_port_count", "Distinct uncommon destination ports"},
	{"port_entropy", "Port distribution entropy (diversity)"},
	{"packet_rate_slope", "Change in event rate vs previous window"},
}

type windowSample struct {
	ts       time.Time
	features models.AnomalyFeatures
}

type windowAccumulator struct {
	start         time.Time
	totalEvents   int
	dnsEvents     int
	httpEvents    int
	tlsEvents     int
	tcpSynEvents  int
	uniqueDevices map[string]struct{}
	ports         map[uint16]int
}

type anomalyDetector struct {
	mu             sync.RWMutex
	window         time.Duration
	baselineNeeded int
	threshold      float64
	maxHistory     int
	maxAlerts      int
	cur            windowAccumulator
	history        []windowSample
	baseline       []models.AnomalyFeatures
	alerts         []models.AnomalyAlert
	latest         models.AnomalySnapshot
	lastRate       float64
}

func newAnomalyDetector() *anomalyDetector {
	now := time.Now()
	return &anomalyDetector{
		window:         30 * time.Second,
		baselineNeeded: 20,
		threshold:      3.5,
		maxHistory:     120,
		maxAlerts:      100,
		cur: windowAccumulator{
			start:         now,
			uniqueDevices: make(map[string]struct{}),
			ports:         make(map[uint16]int),
		},
		latest: models.AnomalySnapshot{
			Status:        "warming_up",
			WindowSeconds: int((30 * time.Second).Seconds()),
		},
	}
}

func (ad *anomalyDetector) observe(now time.Time, evt *models.NetworkEvent, deviceMAC string) {
	ad.mu.Lock()
	defer ad.mu.Unlock()
	if ad.cur.start.IsZero() {
		ad.cur.start = now
	}
	if now.Sub(ad.cur.start) >= ad.window {
		ad.finalizeLocked(now)
		ad.cur = windowAccumulator{
			start:         now,
			uniqueDevices: make(map[string]struct{}),
			ports:         make(map[uint16]int),
		}
	}
	ad.cur.totalEvents++
	switch evt.EventType {
	case models.EVENT_TYPE_DNS:
		ad.cur.dnsEvents++
	case models.EVENT_TYPE_HTTP:
		ad.cur.httpEvents++
	case models.EVENT_TYPE_TLS:
		ad.cur.tlsEvents++
	case models.EVENT_TYPE_TCP:
		if evt.TCPFlags&0x02 != 0 && evt.TCPFlags&0x10 == 0 {
			ad.cur.tcpSynEvents++
		}
	}
	ad.cur.uniqueDevices[deviceMAC] = struct{}{}
	if evt.DstPort > 0 {
		ad.cur.ports[evt.DstPort]++
	}
}

func (ad *anomalyDetector) finalizeLocked(now time.Time) {
	windowSec := now.Sub(ad.cur.start).Seconds()
	if windowSec <= 0 || ad.cur.totalEvents == 0 {
		return
	}
	packetRate := float64(ad.cur.totalEvents) / windowSec
	dnsRate := float64(ad.cur.dnsEvents) / windowSec
	httpRate := float64(ad.cur.httpEvents) / windowSec
	tlsRate := float64(ad.cur.tlsEvents) / windowSec
	synRate := float64(ad.cur.tcpSynEvents) / windowSec
	unusualPorts := countUnusualPorts(ad.cur.ports)
	entropy := portEntropy(ad.cur.ports, ad.cur.totalEvents)
	slope := packetRate - ad.lastRate
	ad.lastRate = packetRate

	features := models.AnomalyFeatures{
		PacketRate:        packetRate,
		DNSRate:           dnsRate,
		HTTPRate:          httpRate,
		TLSRate:           tlsRate,
		TCPSynRate:        synRate,
		UniqueDeviceCount: float64(len(ad.cur.uniqueDevices)),
		UnusualPortCount:  float64(unusualPorts),
		PortEntropy:       entropy,
		PacketRateSlope:   slope,
	}
	ad.history = append(ad.history, windowSample{ts: now, features: features})
	if len(ad.history) > ad.maxHistory {
		ad.history = ad.history[len(ad.history)-ad.maxHistory:]
	}

	status := "warming_up"
	score := 0.0
	z := 0.0
	cent := 0.0
	isAnomaly := false
	var contributions []models.AnomalyContribution
	if len(ad.baseline) < ad.baselineNeeded {
		ad.baseline = append(ad.baseline, features)
	} else {
		status = "active"
		z, contributions = robustZAggregateAndContributions(ad.baseline, features)
		cent = centroidDistance(ad.baseline, features)
		centNorm := math.Min(cent/12.0, 10.0)
		score = (0.72 * z) + (0.28 * centNorm)
		isAnomaly = score >= ad.threshold
		if isAnomaly {
			summary := buildAnomalySummary(contributions)
			ad.alerts = append(ad.alerts, models.AnomalyAlert{
				ObservedAt:    now,
				Score:         score,
				Severity:      severityFromScore(score),
				Reason:        "30s traffic window differs from the learned normal profile (robust z + centroid distance).",
				Summary:       summary,
				Features:      features,
				Contributions: contributions,
			})
			if len(ad.alerts) > ad.maxAlerts {
				ad.alerts = ad.alerts[len(ad.alerts)-ad.maxAlerts:]
			}
		}
	}

	lastSummary := ""
	switch {
	case status == "warming_up":
		lastSummary = "Collecting baseline windows; scoring starts after enough history."
	case len(contributions) > 0 && isAnomaly:
		lastSummary = buildAnomalySummary(contributions)
	case len(contributions) > 0 && !isAnomaly:
		top := contributions[0]
		lastSummary = fmt.Sprintf(
			"No anomaly (score %.2f < %.1f). Largest deviation: %s (≈%.1f robust σ from typical).",
			score, ad.threshold, top.Label, top.RobustZ,
		)
	}

	ad.latest = models.AnomalySnapshot{
		WindowSeconds:      int(ad.window.Seconds()),
		Status:             status,
		BaselineWindows:    len(ad.baseline),
		CurrentScore:       score,
		RobustZScore:       z,
		CentroidDistance:   cent,
		IsAnomaly:          isAnomaly,
		LastFeatures:       features,
		LastEvaluatedAt:    now,
		LastSummary:        lastSummary,
		LastContributions:  contributions,
		RecentAnomalyCount: countRecent(ad.alerts, now.Add(-10*time.Minute)),
	}
}

func (ad *anomalyDetector) status() models.AnomalySnapshot {
	ad.mu.RLock()
	defer ad.mu.RUnlock()
	out := ad.latest
	out.RecentAlerts = make([]models.AnomalyAlert, len(ad.alerts))
	copy(out.RecentAlerts, ad.alerts)
	if len(out.RecentAlerts) > 20 {
		out.RecentAlerts = out.RecentAlerts[len(out.RecentAlerts)-20:]
	}
	sort.Slice(out.RecentAlerts, func(i, j int) bool {
		return out.RecentAlerts[i].ObservedAt.After(out.RecentAlerts[j].ObservedAt)
	})
	return out
}

func severityFromScore(score float64) string {
	if score >= 6 {
		return "high"
	}
	if score >= 4.5 {
		return "medium"
	}
	return "low"
}

func countRecent(alerts []models.AnomalyAlert, since time.Time) int {
	n := 0
	for _, a := range alerts {
		if a.ObservedAt.After(since) {
			n++
		}
	}
	return n
}

func countUnusualPorts(ports map[uint16]int) int {
	common := map[uint16]struct{}{
		22: {}, 53: {}, 67: {}, 68: {}, 80: {}, 123: {}, 443: {}, 8080: {}, 8443: {}, 853: {},
	}
	n := 0
	for p := range ports {
		if _, ok := common[p]; !ok {
			n++
		}
	}
	return n
}

func portEntropy(ports map[uint16]int, total int) float64 {
	if total == 0 {
		return 0
	}
	h := 0.0
	for _, c := range ports {
		p := float64(c) / float64(total)
		if p > 0 {
			h -= p * math.Log2(p)
		}
	}
	return h
}

func featureVector(f models.AnomalyFeatures) []float64 {
	return []float64{
		f.PacketRate, f.DNSRate, f.HTTPRate, f.TLSRate, f.TCPSynRate,
		f.UniqueDeviceCount, f.UnusualPortCount, f.PortEntropy, f.PacketRateSlope,
	}
}

// robustZAggregateAndContributions returns the mean capped per-feature robust z-score
// plus per-feature rows for UI explanations. MAD is floored relative to the median so
// near-constant baselines do not produce astronomical z-scores.
func robustZAggregateAndContributions(baseline []models.AnomalyFeatures, cur models.AnomalyFeatures) (float64, []models.AnomalyContribution) {
	curV := featureVector(cur)
	contributions := make([]models.AnomalyContribution, 0, len(curV))
	sum := 0.0
	for i := range curV {
		vals := make([]float64, 0, len(baseline))
		for _, b := range baseline {
			vals = append(vals, featureVector(b)[i])
		}
		med := median(vals)
		mad := medianAbsoluteDeviation(vals, med)
		scaledMAD := math.Max(mad, 0.12*math.Max(math.Abs(med), 0.25))
		if scaledMAD < 1e-6 {
			scaledMAD = 1e-6
		}
		rawZ := math.Abs((curV[i] - med) / (1.4826 * scaledMAD))
		capped := math.Min(rawZ, perFeatureZCap)
		sum += capped
		meta := anomalyFeatureMeta[i]
		contributions = append(contributions, models.AnomalyContribution{
			Feature:        meta.Key,
			Label:          meta.Label,
			Value:          curV[i],
			BaselineMedian: med,
			RobustZ:        capped,
		})
	}
	sort.Slice(contributions, func(i, j int) bool {
		return contributions[i].RobustZ > contributions[j].RobustZ
	})
	return sum / float64(len(curV)), contributions
}

func buildAnomalySummary(c []models.AnomalyContribution) string {
	if len(c) == 0 {
		return ""
	}
	n := 3
	if len(c) < n {
		n = len(c)
	}
	parts := make([]string, 0, n)
	for i := 0; i < n; i++ {
		parts = append(parts, fmt.Sprintf(
			"%s is about %.1f robust σ from typical (baseline median %.2f, this window %.2f)",
			c[i].Label, c[i].RobustZ, c[i].BaselineMedian, c[i].Value,
		))
	}
	return strings.Join(parts, "; ") + "."
}

func centroidDistance(baseline []models.AnomalyFeatures, cur models.AnomalyFeatures) float64 {
	if len(baseline) == 0 {
		return 0
	}
	dims := len(featureVector(cur))
	cent := make([]float64, dims)
	for _, b := range baseline {
		v := featureVector(b)
		for i := range v {
			cent[i] += v[i]
		}
	}
	for i := range cent {
		cent[i] /= float64(len(baseline))
	}
	curV := featureVector(cur)
	sumSq := 0.0
	for i := range curV {
		d := curV[i] - cent[i]
		sumSq += d * d
	}
	return math.Sqrt(sumSq)
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	cp := append([]float64(nil), values...)
	sort.Float64s(cp)
	m := len(cp) / 2
	if len(cp)%2 == 0 {
		return (cp[m-1] + cp[m]) / 2
	}
	return cp[m]
}

func medianAbsoluteDeviation(values []float64, med float64) float64 {
	if len(values) == 0 {
		return 0
	}
	dev := make([]float64, 0, len(values))
	for _, v := range values {
		dev = append(dev, math.Abs(v-med))
	}
	return median(dev)
}
