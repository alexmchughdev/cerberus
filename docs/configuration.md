# Configuration (environment)

Cerberus reads the following environment variables at runtime.

| Variable | Default | Effect |
|----------|---------|--------|
| `CERBERUS_HTTP_ADDR` | `127.0.0.1:8080` | Listen address for REST, `/metrics`, and the Control Room. Use `0.0.0.0:8080` when exposing the dashboard from Docker or a LAN. |
| `CERBERUS_GEOIP_DB` | *(unset)* | Path to a MaxMind **GeoLite2-City.mmdb** (or compatible) file. When set and loadable, device records gain optional geo fields for public IPs. |
| `CERBERUS_DATA_DIR` | `./data` | Directory for BuntDB file (see `cmd/cerberus`), IEEE OUI cache, IANA services CSV cache, and related files. |
| `CERBERUS_DB_ONLINE` | *(unset)* | When `1`, `true`, or `yes`, allows automatic download/refresh of IEEE OUI and IANA service registries when local cache is missing or stale. |

Interface selection and LRU size are currently set in code (`cmd/cerberus` / `monitor.NewNetworkMonitor`); see root README “Configuration” for pointers.
