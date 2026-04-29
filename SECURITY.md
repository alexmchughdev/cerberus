# Security Policy

## Supported versions

Only the latest release is actively maintained.

## Reporting a vulnerability

Do not open a public issue for security vulnerabilities.

Email: **med@zrouga.email** (or use GitHub's private vulnerability reporting under Security → Advisories).

Include:
- Description of the issue
- Steps to reproduce
- Kernel version and distro
- Potential impact

You'll get an acknowledgment within 72 hours. If the issue is confirmed, a fix will be prioritized before any public disclosure.

## Scope

Cerberus runs as root and attaches eBPF programs to TC hooks. The relevant attack surfaces are:

- The eBPF object file (`cerberus_tc.o`) being replaced or tampered with before load
- The REST API (`127.0.0.1:8080` by default) being exposed without access controls
- The BuntDB file (`./data/network.db`) containing observed network metadata

The tool captures packet metadata and the first 128 bytes of payload for L7 inspection. It does **not** store full packet payloads.