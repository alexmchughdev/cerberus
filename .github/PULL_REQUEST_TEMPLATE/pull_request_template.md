## What this does

<!-- One paragraph. What changed and why. -->

## Type

- [ ] Bug fix
- [ ] New protocol / detection
- [ ] Alert rule / anomaly
- [ ] API / dashboard
- [ ] Docs
- [ ] CI / build

## Checklist

- [ ] `make all` passes locally
- [ ] `go vet ./...` and `gofmt` clean
- [ ] If touching `cerberus_tc.c` — struct sizes match `models.EventSize`
- [ ] If adding an alert rule — added to `docs/threat-and-anomaly-patterns.md`
- [ ] If changing the API — updated `docs/api-reference.md`

## Testing

<!-- How did you verify this. `sudo ./cerberus`, specific traffic generated, distro tested on. -->

## Related issues

<!-- Closes #N -->