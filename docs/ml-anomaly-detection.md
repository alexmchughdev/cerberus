# Anomaly detection (“ML-lite”)

Cerberus does **not** ship a neural network or external ML runtime. The monitor includes a small **unsupervised anomaly detector** that treats each time window as a **feature vector**, learns a **robust normal profile** from the first windows after startup, then flags windows whose features deviate strongly. The UI and API refer to this as **ML-lite** because it behaves like a lightweight behavioral model: score, severity, and **human-readable explanations** per feature.

Code lives in `internal/monitor/anomaly.go`; types are in `internal/models` (`AnomalyFeatures`, `AnomalyContribution`, `AnomalyAlert`, `AnomalySnapshot`). The detector runs inside `TrackEvent` via `anomaly.observe` (same process as traffic ingestion).

---

## 1. What problem it solves

On a busy LAN you want to notice **bulk changes in traffic shape** (rate spikes, many rare ports, sudden SYN storms) without hand-tuned thresholds for every metric. The detector summarizes each interval into a fixed set of rates and diversity measures, compares them to a **learned baseline**, and emits an alert when a **combined score** crosses a threshold.

It complements **rule-based alerts** (DNS/TCP volume, unique targets), which use simple thresholds on per-device counters. Anomaly detection is **global per process**: one sliding view of “what the network looked like in the last 30 seconds,” not per-device classification.

---

## 2. Time windows and lifecycle

| Concept | Value (current code) | Meaning |
|--------|----------------------|--------|
| **Window length** | 30 seconds | Events are accumulated into `windowAccumulator`; when `now - window_start ≥ 30s`, the window is **finalized** and a feature vector is computed. |
| **Baseline collection** | First **20** finalized windows | Each window’s features are **appended** to `baseline`. During this phase `status` is `warming_up` and **no anomaly score** is produced for alerting. |
| **Active scoring** | After 20 baseline windows exist | Each new finalized window is compared to the **fixed** baseline (those 20 vectors). The baseline **does not** keep updating with new traffic after warm-up (see [Limitations](#8-limitations-and-caveats)). |
| **History ring** | Up to **120** `windowSample` entries | Used internally; baseline for scoring remains the initial 20 feature vectors. |
| **Stored alerts** | Up to **100** `AnomalyAlert` rows | Older entries are dropped from the tail. |
| **API / UI recent list** | Up to **20** alerts in the snapshot | Newest-first after copy. |

Warm-up copy in the snapshot: *“Collecting baseline windows; scoring starts after enough history.”* Once active, non-anomaly windows can still get a short *“No anomaly (score …). Largest deviation: …”* summary.

---

## 3. Features (the vector)

Each finalized window becomes an `AnomalyFeatures` struct. All of these are **derived inside the detector** from counts in the current window (and the previous packet rate for slope):

| JSON field | Source intuition |
|------------|-------------------|
| `packet_rate` | Total events in the window ÷ window duration (events/sec). |
| `dns_rate` | DNS event count ÷ duration. |
| `http_rate` | HTTP event count ÷ duration. |
| `tls_rate` | TLS event count ÷ duration. |
| `tcp_syn_rate` | TCP segments with SYN set and ACK clear ÷ duration. |
| `unique_device_count` | Number of distinct source MACs seen in the window. |
| `unusual_port_count` | Count of **distinct** destination ports that are **not** in a small “common” set (22, 53, 67, 68, 80, 123, 443, 8080, 8443, 853). |
| `port_entropy` | Shannon entropy (base 2) over the **distribution** of destination ports in the window (diversity of where traffic goes). |
| `packet_rate_slope` | Current window `packet_rate` minus **previous** window’s `packet_rate` (step change signal). |

The **feature vector** used for math is exactly these nine numbers in a fixed order (`featureVector` in code).

---

## 4. Baseline and robust deviation

For each feature dimension separately:

1. Collect the values of that dimension across all **baseline** windows (the stored 20 vectors).
2. Compute the **median** of those values.
3. Compute **MAD** (median absolute deviation) around that median, then scale similarly to a robust σ (division by `1.4826 × scaledMAD` in code).
4. **Scale MAD** with a floor (`scaledMAD = max(MAD, 0.12 × max(|median|, 0.25))`) so near-constant baselines do not produce huge z-scores.
5. **Robust z** for the current window: absolute deviation from median, divided by the scaled spread, then **capped** at `perFeatureZCap` (8.0) per feature.

The **aggregate robust z** exposed as `robust_z_score` is the **mean** of these nine capped per-feature z values (not a literal single z-test on one scalar).

**Contributions** (`AnomalyContribution`): for each feature you get `value`, `baseline_median`, and `robust_z` (capped), plus a stable `label` string for the UI (e.g. “Events per second (all types)”).

---

## 5. Centroid distance and combined score

- **Centroid**: mean of the nine-dimensional baseline vectors (one mean per dimension).
- **Centroid distance**: Euclidean distance from the **current** feature vector to that centroid (`sqrt(sum of squared diffs)`).
- **Normalization**: `centNorm = min(centroidDistance / 12.0, 10.0)` (dimensionless cap).
- **Combined score**:  
  `score = 0.72 × (mean capped robust z) + 0.28 × centNorm`

So the model blends **per-feature robust outliers** with **joint drift** away from the “average normal window” in feature space.

---

## 6. When is it an anomaly?

- **Threshold** (fixed in code): `score >= 3.5` ⇒ `is_anomaly` is true for that window.
- **Severity** from `score`:  
  - `high` if score ≥ 6  
  - `medium` if score ≥ 4.5  
  - `low` otherwise  

When an anomaly fires, an `AnomalyAlert` is appended with:

- `reason`: fixed explanation string about the 30s window vs learned profile.
- `summary`: **plain-language** lead (“This looks unusual mainly because …”) built from the top contributing features.
- `detail`: optional **technical** paragraph (medians, robust σ per line) for operators who want numbers—mirrors what used to be the only `summary` field.

The snapshot exposes the same split: `last_summary` (plain) and `last_summary_detail` (technical). The Control Room shows plain text first and puts the technical paragraph plus per-feature lines under **“Technical details behind the reasoning”**.

When it does **not** fire, `last_summary` explains that the score stayed below threshold; `last_summary_detail` still names the strongest numerical deviation.

---

## 7. API and UI

- **HTTP**: `GET /api/v1/anomalies` returns an `AnomalySnapshot` JSON object (`window_seconds`, `status`, `baseline_windows`, `current_score`, `robust_z_score`, `centroid_distance`, `is_anomaly`, `last_features`, `last_evaluated_at`, `last_summary`, `last_contributions`, `recent_anomaly_count`, `recent_alerts`).
- **Control Room**: Overview shows a short anomaly block; the **Anomalies** route shows full detail, contributor table, and alert history. See [web-ui.md](web-ui.md).

The snapshot is read under the detector’s own mutex; it does not require holding the main monitor lock for the anomaly struct (alerts and `latest` are updated from `observe` while holding `anomalyDetector.mu`).

---

## 8. Limitations and caveats

- **Not deep learning**: no training loop, GPU, or external model files.
- **Frozen baseline after warm-up**: the baseline is the **first 20 windows only**. Long-term drift (e.g. daytime vs nighttime) is not continuously re-learned; restarting the process starts a new baseline.
- **Global, not per-subnet**: one detector per monitor process; it does not isolate a single attacker IP by design.
- **SYN floods and heavy scans** can push rates, port entropy, and unusual-port counts; combined with UI polling, ensure you are on a build that **snapshots** device maps for the API to avoid races (see [system-overview.md](system-overview.md) concurrency note).
- **Tuning**: window length, baseline count, threshold, and score weights are **constants in Go** today, not environment variables.

---

## 9. Related code and docs

| Item | Location |
|------|----------|
| Detector implementation | `internal/monitor/anomaly.go` |
| Types | `internal/models/types.go` (`Anomaly*`) |
| Wiring | `internal/monitor/monitor.go` (`newAnomalyDetector`, `GetAnomalySnapshot`, `TrackEvent` → `anomaly.observe`) |
| HTTP handler | `internal/api/server.go` (`handleAnomalies`) |
| Tests | `internal/monitor/anomaly_test.go` |

Related reading: [system-overview.md](system-overview.md), [api-reference.md](api-reference.md), [web-ui.md](web-ui.md). For SYN/flood/scan-style interpretations of these features (and honest limits), see [threat-and-anomaly-patterns.md](threat-and-anomaly-patterns.md).
