<div align="center">

# S3eker

Real‑time terminal toolkit for open bucket discovery and Firebase configuration checks.

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Status](https://img.shields.io/badge/status-beta-yellow)](#-roadmap)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](#-contributing)

</div>

---

## What’s inside
- Interactive launcher with colorful CLI and ASCII banner
- Bucket scraper (TUI): streams results, live filters, adjustable thresholds
- Firebase checker (wizard): probes auth/RTDB/Storage/Firestore, prints a formatted table, optional Markdown export

> Default scraping target is GrayhatWarfare Random Buckets. You can set a custom URL.

## Quickstart

Build from source:
```bash
# Build all binaries with one command
./build.sh

# Run the main launcher
./s3eker
```

Or build manually:
```bash
go build -o s3eker ./cmd/launcher
go build -o s3eker-scraper ./cmd/scraper
go build -o s3eker-fbscan ./cmd/fbscan
./s3eker
```

Launcher options:
1) Scrape Grayhat for new open buckets (CLI)
2) Check Firebase configuration (wizard)
3) Close

The wizard accepts a `GoogleService-Info.plist` or prompts for API key, project ID, database URL, and storage bucket.

Example Firebase run (wizard):
```
✔ AnonymousAuth       PASS   status=404
✔ SignUp              PASS   status=400
✗ RTDBPublicRead      FAIL   /.json readable (200)
✗ StoragePublicList   FAIL   listable (200)
✗ StorageWrite        FAIL   write allowed (200)
✔ StorageDelete       PASS   status=204
```

Exports:
- JSON report saved to the chosen `-out`
- Optional Markdown summary (`.md`) right next to the JSON

## Flags (advanced)
```text
./s3eker-fbscan -fb-plist /path/GoogleService-Info.plist -out fb_report.json
  -fb-api-key string         Firebase API key
  -fb-project-id string      Firebase project id
  -fb-rtdb-url string        Firebase Realtime Database URL
  -fb-storage-bucket string  Firebase Storage bucket (e.g., myapp.appspot.com)
  -fb-plist string           Path to GoogleService-Info.plist / Info.plist
  -out string                Output report file (JSON)
```

Global UX flags (launcher will auto-detect):
- `NO_COLOR=1` to disable colors
- Non‑TTY output switches to plain ASCII and minimal formatting automatically

## Build & Install

### Quick Build (Recommended)
```bash
# Build all binaries with one command
./build.sh

# Run from the cloned directory
./s3eker
```

### Manual Build
```bash
go build -o s3eker ./cmd/launcher
go build -o s3eker-scraper ./cmd/scraper
go build -o s3eker-fbscan ./cmd/fbscan
./s3eker
```

### System-wide Installation (Optional)
If you want to install the binaries system-wide so you can run `s3eker` from anywhere:
```bash
# Build first
./build.sh

# Install to /usr/local/bin
sudo install -m 0755 s3eker /usr/local/bin/s3eker
sudo install -m 0755 s3eker-scraper /usr/local/bin/s3eker-scraper
sudo install -m 0755 s3eker-fbscan /usr/local/bin/s3eker-fbscan
```

## Exit codes
- `0` no failures
- `2` warnings only
- `3` failures present (useful for CI gates)

## Troubleshooting
- No results from scraper? Lower threshold (`m`), confirm HTTP code progress updates.
- If upstream HTML changes, selectors may need small updates.
- For Firebase probes, network errors are shown as INFO and do not stop other probes.

## Roadmap
- Pluggable sources (GCS/Azure listings)
- CSV/NDJSON exporters
- Rules fingerprinting and hints
- Settings file (`~/.config/s3eker/config.yaml`) for theme/output defaults

## Contributing
PRs welcome. Keep diffs focused and add a brief before/after note. Run `go build` locally.

## Ethics & Legal
Use only with authorization. Do not access or retain sensitive data. You are responsible for legal compliance.

---

Made with Go. Feedback and ideas welcome.