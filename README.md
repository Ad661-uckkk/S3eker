# S3eker

<div align="center">

Real‑time terminal UI (TUI) for discovering public cloud buckets. Streams results live while you scrape, deduplicates on the fly, and persists to JSON.

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Status](https://img.shields.io/badge/status-beta-yellow)](#-roadmap)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](#-contributing)

</div>

---

### Why S3eker?
Investigating open buckets across providers often involves brittle scripts and poor feedback loops. S3eker gives you a responsive terminal interface, a steady stream of findings, and guardrails to keep results meaningful and reproducible.

## ✨ Features
- Live TUI with continuously streaming results
- Adjustable minimum file threshold (drop noise, keep signal)
- Runtime configuration (change source URL without restart)
- Automatic in‑memory deduplication and JSON persistence
- CLI mode for headless servers and automation

> Note: By default S3eker targets GrayhatWarfare’s Random Buckets page. You can point it at another compatible listing with `-url`.

## 📦 Installation

Build from source:
```bash
go build -o s3eker .
sudo mv s3eker /usr/local/bin/
```

Run without install:
```bash
go run .
```

## 🚀 Quick Start

### GUI (default)
```bash
s3eker
```
Key bindings:
- `q` Quit
- `u` Set source URL
- `m` Set minimum file count

The status bar shows: host, pages fetched, new/total buckets, last HTTP status, error count, and the current `min` threshold. It refreshes every second.

### CLI / Headless
```bash
s3eker -gui=false \
  -url="https://buckets.grayhatwarfare.com/random/buckets" \
  -min=500 \
  -o merged_deduplicated.json
```

Flags:
- `-url` string: Source list page (default: random buckets)
- `-min` int: Minimum file count to include (default: 1000)
- `-o` string: Output JSON file (default: `merged_deduplicated.json`)
- `-gui` bool: Enable TUI (default: true)

Output is written to `merged_deduplicated.json` in the working directory. This file and any sample data under `buckets/` are intentionally excluded from version control for public releases.

## 🧰 Troubleshooting
- Seeing no results? Lower the threshold with `m` (GUI) or `-min` (CLI) and watch the status counters to verify pages are fetched.
- If the upstream HTML changes, selectors may need minor tweaks.
- Respect provider rate limits and terms of service.

## 🗺 Roadmap
- Pluggable sources (multi‑provider listings)
- Export formats (CSV/NDJSON)
- Rule‑based highlighting (keywords, domains)
- Optional disk‑backed deduplication cache

## 🤝 Contributing
Contributions welcome! If you have an idea or find a bug:
- Open an issue describing the problem or proposal
- Submit a PR with a clear description and minimal diff

Before committing, run `go build` (and any added tests) to ensure things compile cleanly.

## 🔐 Ethics & Legal
Use only where you have authorization. Do not access, download, or distribute sensitive data. You are fully responsible for complying with all applicable laws, terms, and policies.

---

Made with Go. If this project helps you, consider sharing feedback or ideas for the roadmap.