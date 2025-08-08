# S3eker (Beta)

Real-time terminal UI (TUI) that streams public cloud buckets as they are discovered from GrayhatWarfare’s Random Buckets page. Buckets are deduplicated and appended to a local JSON file.

## Features
- Live TUI with streaming results
- Adjustable minimum file threshold
- Change source URL at runtime
- Automatic deduplication and JSON persistence

## Install

Build locally:
```bash
go build -o s3eker .
sudo mv s3eker /usr/local/bin/
```

Or run in-place:
```bash
go run .
```

## Usage

### GUI (default)
```bash
s3eker
```
Keys:
- q: quit
- u: set source URL
- m: set minimum file count

Status bar shows URL host, pages fetched, new/total buckets, last HTTP status, errors, and min threshold.

### CLI
```bash
s3eker -gui=false -url="https://buckets.grayhatwarfare.com/random/buckets" -min=500 -o merged_deduplicated.json
```
Flags:
- `-url` string: source list page (default random buckets)
- `-min` int: minimum file count to include (default 1000)
- `-o` string: output JSON file (default merged_deduplicated.json)
- `-gui` bool: enable TUI (default true)

Output is written to `merged_deduplicated.json` in the working directory. This file and any sample data under `buckets/` are ignored by git for public releases.

## Notes
- Scraper relies on static HTML table at GrayhatWarfare; selectors may need updates if the site changes.
- Respect rate limits and terms of service.

## Legal / Ethics
Use only where you have permission. Do not access, download, or distribute sensitive data. You are responsible for complying with applicable laws.