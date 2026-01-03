package api

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"sort"
	"time"

	"github.com/zrougamed/cerberus/internal/models"
	"github.com/zrougamed/cerberus/internal/monitor"
)

//go:embed web/*
var webFS embed.FS

type Server struct {
	monitor *monitor.NetworkMonitor
}

func NewServer(mon *monitor.NetworkMonitor) *Server {
	return &Server{monitor: mon}
}

func (s *Server) Handler() (http.Handler, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/summary", s.handleSummary)
	mux.HandleFunc("/api/v1/devices", s.handleDevices)

	staticSub, err := fs.Sub(webFS, "web")
	if err != nil {
		return nil, err
	}
	mux.Handle("/", http.FileServer(http.FS(staticSub)))

	return mux, nil
}

type summaryResponse struct {
	GeneratedAt  time.Time           `json:"generated_at"`
	PacketStats  map[string]uint64   `json:"packet_stats"`
	DeviceCount  int                 `json:"device_count"`
	TopServices  map[string]int      `json:"top_services"`
	TopVendors   map[string]int      `json:"top_vendors"`
	RecentDevice []deviceSummaryItem `json:"recent_devices"`
}

type deviceSummaryItem struct {
	MAC          string    `json:"mac"`
	IP           string    `json:"ip"`
	Vendor       string    `json:"vendor"`
	FirstSeen    time.Time `json:"first_seen"`
	LastSeen     time.Time `json:"last_seen"`
	TCP          int       `json:"tcp"`
	UDP          int       `json:"udp"`
	DNSQueries   int       `json:"dns_queries"`
	HTTPRequests int       `json:"http_requests"`
	TLS          int       `json:"tls"`
}

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	devices := s.monitor.GetStats()
	packetStats := map[string]uint64{
		"total": s.monitor.Stats.TotalPackets,
		"arp":   s.monitor.Stats.ArpPackets,
		"tcp":   s.monitor.Stats.TcpPackets,
		"udp":   s.monitor.Stats.UdpPackets,
		"icmp":  s.monitor.Stats.IcmpPackets,
		"dns":   s.monitor.Stats.DnsPackets,
		"http":  s.monitor.Stats.HttpPackets,
		"tls":   s.monitor.Stats.TlsPackets,
	}

	topServices := make(map[string]int)
	topVendors := make(map[string]int)
	deviceItems := make([]deviceSummaryItem, 0, len(devices))

	for _, d := range devices {
		for svc, count := range d.Services {
			topServices[svc] += count
		}
		topVendors[d.Vendor]++
		deviceItems = append(deviceItems, deviceSummaryItem{
			MAC:          d.MAC,
			IP:           d.IP,
			Vendor:       d.Vendor,
			FirstSeen:    d.FirstSeen,
			LastSeen:     d.LastSeen,
			TCP:          d.TCPConnections,
			UDP:          d.UDPConnections,
			DNSQueries:   d.DNSQueries,
			HTTPRequests: d.HTTPRequests,
			TLS:          d.TLSConnections,
		})
	}

	sort.Slice(deviceItems, func(i, j int) bool {
		return deviceItems[i].LastSeen.After(deviceItems[j].LastSeen)
	})
	if len(deviceItems) > 8 {
		deviceItems = deviceItems[:8]
	}

	writeJSON(w, http.StatusOK, summaryResponse{
		GeneratedAt:  time.Now(),
		PacketStats:  packetStats,
		DeviceCount:  len(devices),
		TopServices:  topMap(topServices, 8),
		TopVendors:   topMap(topVendors, 8),
		RecentDevice: deviceItems,
	})
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	devices := s.monitor.GetStats()
	out := make([]*models.DeviceInfo, 0, len(devices))
	for _, d := range devices {
		out = append(out, d)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].LastSeen.After(out[j].LastSeen)
	})

	writeJSON(w, http.StatusOK, out)
}

func topMap(values map[string]int, limit int) map[string]int {
	type pair struct {
		key   string
		value int
	}
	items := make([]pair, 0, len(values))
	for k, v := range values {
		items = append(items, pair{key: k, value: v})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].value > items[j].value
	})
	if len(items) > limit {
		items = items[:limit]
	}
	result := make(map[string]int, len(items))
	for _, item := range items {
		result[item.key] = item.value
	}
	return result
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
