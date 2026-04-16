# Deployment footprint and throughput

Cerberus is designed so **most work stays in the kernel**: the TC eBPF program parses packets, emits small fixed-size events into a ring buffer, and userspace aggregates them. You should see **low CPU and memory** in the container (or host process) even when the **observed traffic rate** on the wire is high.

## Example: `docker stats` under load

Typical numbers vary with interface count, cache size, and whether the Control Room is open, but the order of magnitude is small. One real capture while the network was busy looked like this:

| CONTAINER ID | NAME     | CPU % | MEM USAGE / LIMIT | MEM % | NET I/O | BLOCK I/O     | PIDS |
|--------------|----------|-------|-------------------|-------|---------|---------------|------|
| fb0229057224 | cerberus | 0.59% | 60.4MiB / 15.35GiB | 0.38% | 0B / 0B | 4.1kB / 13.8MB | 15 |

**NET I/O** for the container often stays **0B / 0B** because the process does not proxy traffic: it attaches eBPF programs to host interfaces and reads metadata from the kernel. Traffic does not flow *through* the container network namespace in the usual “reverse proxy” sense.

**BLOCK I/O** reflects BuntDB and optional cache files (for example `./data` volume), not per-packet disk writes.

## Why usage stays low at high throughput

- **No full packet capture**: only metadata plus a short L7 preview (see root README security notes).
- **Ring buffer + batching**: events are fixed-size; userspace drains the buffer and updates in-memory structures.
- **LRU device cache**: bounds the number of live `DeviceInfo` records held in RAM (configured at monitor construction).
- **Periodic persistence**: BuntDB writes are not per-packet.

If CPU or RSS grows unexpectedly, check **number of attached interfaces**, **dashboard poll rate** (many concurrent JSON snapshots), **`CERBERUS_DB_ONLINE`** refreshes, and **GeoIP** loading—all are optional or configurable.

## Related

- [configuration.md](configuration.md) — `CERBERUS_HTTP_ADDR`, `CERBERUS_DATA_DIR`, etc.
- [system-overview.md](system-overview.md) — data path from eBPF to API.
