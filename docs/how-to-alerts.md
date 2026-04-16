# How to trigger and use alerts

Cerberus exposes **two kinds** of alerts in the API and Control Room:

| Kind | Where in UI | Endpoint | What fires them |
|------|-------------|----------|------------------|
| **Rule-based** | **Rule alerts** | `GET /api/v1/alerts` | Per-device thresholds on DNS count, TCP connection count, and unique target count |
| **Anomaly (ML-lite)** | **Anomalies** | `GET /api/v1/anomalies` | Global 30-second windows vs a learned baseline (combined score ≥ threshold) |

---

## 1. Rule-based alerts (`AlertEvent`)

Evaluation runs whenever a device is updated (`evaluateAlerts` in `internal/monitor`). Defaults are set in `NewNetworkMonitor` (not environment variables today):

| Rule ID | Condition | Default threshold |
|---------|-----------|---------------------|
| `dns_query_volume` | `DNSQueries` for that MAC | **> 200** |
| `tcp_connection_volume` | `TCPConnections` for that MAC | **> 500** |
| `target_spread` | `len(Targets)` — rolling list of **up to 20** unique destination IPs per device | **> 18** (i.e. **19+** distinct IPs seen in the windowed list) |

**Deduplication:** For each `(device MAC, rule)` pair, Cerberus emits **one** alert the first time the condition becomes true. It does **not** spam a new row on every packet while the condition stays true. When the condition **clears** (counts fall back under the threshold), the internal latch resets; crossing again can produce a **new** alert.

### How to trigger them (testing / demos)

1. **`dns_query_volume`**  
   From one host on the monitored LAN, generate **more than 200 DNS queries** while Cerberus is running (e.g. scripted `dig`/`nslookup` in a loop to many names). Easiest if that host is the **same MAC** Cerberus attributes traffic to.

2. **`tcp_connection_volume`**  
   Open **more than 500 TCP connections** from one source MAC (e.g. many parallel `curl`/`nc` to different ports or hosts). Tools like connection stress tests or port scanners can hit this on a busy client.

3. **`target_spread`**  
   From one MAC, connect toward **19+ different destination IPs** (Cerberus keeps a rolling list of the last **20** unique `dstIP` values per device). Port scans or many short parallel curls to different addresses can trigger this once the list fills past the threshold.

**View:** Control Room → **Rule alerts**, or `curl -s http://127.0.0.1:8080/api/v1/alerts` (adjust bind address if needed).

**Lowering thresholds for lab use:** Change `alertConfig` in `internal/monitor/monitor.go` inside `NewNetworkMonitor` (e.g. `MaxDNSQueriesPerDevice: 50`), rebuild, and rerun.

---

## 2. Anomaly alerts (`AnomalyAlert`)

The detector needs **20 completed 30-second baseline windows** (~10 minutes of operation) before it enters **active** scoring. Then each new window gets a **score**; if **score ≥ 3.5**, an anomaly alert is recorded for that window.

### How to trigger them (testing / demos)

- **SYN / port-scan style traffic:** Many SYN packets, high SYN rate, many uncommon destination ports, or sharp jumps in overall event rate vs baseline (e.g. `hping3`, `nmap -sS`, or lab SYN flood tools **only on networks you own**).
- **Volume spikes:** Sudden bulk DNS, HTTP, or mixed traffic that pushes `packet_rate`, per-protocol rates, or **packet_rate_slope** far from the first 20 windows.

**View:** Control Room → **Anomalies**, or `GET /api/v1/anomalies`. Plain-language **summary** and **technical detail** explain which features moved.

**Warm-up:** Until baseline is ready, you will see `warming_up` and no anomaly scores for alerting.

See [threat-and-anomaly-patterns.md](threat-and-anomaly-patterns.md) for what these signals *mean*, and [ml-anomaly-detection.md](ml-anomaly-detection.md) for math and limits.

### SYN floods and DDoS (short version)

- There is **no** separate “SYN flood” or “DDoS” rule ID. Both usually appear as **anomaly alerts** when **`tcp_syn_rate`**, **`packet_rate`**, and related features jump vs baseline (SYN-heavy traffic), or when total volume/spikes dominate the score (DDoS-like conditions).
- Cerberus **does not block** or **mitigate** traffic; it **observes** and **scores**. For nuance (SYN flood vs scan vs distributed flood), read the **SYN flood** and **DDoS** subsections in [threat-and-anomaly-patterns.md](threat-and-anomaly-patterns.md).

---

## 3. Prometheus (optional)

`GET /metrics` exposes packet and device counters. You can **route Prometheus alerts** on those series (e.g. rate of `cerberus_packets_total`) for ops-level paging—outside Cerberus’s built-in alert list.

---

## 4. Quick checklist

| Goal | Action |
|------|--------|
| See rule alerts | Exceed DNS / TCP / target thresholds **from one device**, then open **Rule alerts** |
| See anomaly alerts | Wait for baseline, then generate traffic very different from the first ~10 minutes |
| Reset rule latch | Let counts drop under threshold; next crossing can alert again |
| No alerts at all | Normal quiet traffic; thresholds never crossed; anomaly score stays &lt; 3.5 |

---

## Related

- [api-reference.md](api-reference.md) — JSON shapes  
- [web-ui.md](web-ui.md) — where alerts appear in the Control Room  
- [configuration.md](configuration.md) — `CERBERUS_HTTP_ADDR`, etc.  
