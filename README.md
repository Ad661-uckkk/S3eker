# DigitalOcean Spaces Scanner for Bug Bounty Research

This tool adapts the powerful [S3Scanner](https://github.com/sa7mon/S3Scanner) specifically for DigitalOcean Spaces enumeration in bug bounty research.

## Prerequisites

### 1. Install S3Scanner

First, you need to install S3Scanner. You can either:

**Option A: Download Pre-built Binary**
```bash
# Download latest release from GitHub
wget https://github.com/sa7mon/S3Scanner/releases/latest/download/s3scanner-linux-amd64
chmod +x s3scanner-linux-amd64
mv s3scanner-linux-amd64 s3scanner
sudo mv s3scanner /usr/local/bin/
```

**Option B: Build from Source (requires Go)**
```bash
git clone https://github.com/sa7mon/S3Scanner.git
cd S3Scanner
go build -o s3scanner .
sudo mv s3scanner /usr/local/bin/
```

### 2. Install Python Dependencies
```bash
pip3 install requests
```

## Usage

### Basic Scanning

```bash
# Scan with company-specific wordlist
python3 do-space-scanner.py -c "company-name"

# Scan with custom words
python3 do-space-scanner.py -w "uploads" "assets" "backup"

# Combine company name with custom words
python3 do-space-scanner.py -c "acme" -w "staging" "prod" "api"
```

### Advanced Options

```bash
# Enable object enumeration (time-consuming)
python3 do-space-scanner.py -c "company-name" -e

# Use more threads for faster scanning
python3 do-space-scanner.py -c "company-name" -t 8

# Save detailed report to file
python3 do-space-scanner.py -c "company-name" -o report.txt

# Full scan with all options
python3 do-space-scanner.py -c "company-name" -w "staging" "api" -e -t 8 -o detailed_report.txt
```

### Direct S3Scanner Usage for DigitalOcean

You can also use S3Scanner directly:

```bash
# Create a wordlist file
echo -e "assets\nimages\nuploads\nbackup" > spaces.txt

# Scan with S3Scanner
s3scanner -bucket-file spaces.txt -provider digitalocean -json

# Enable enumeration
s3scanner -bucket-file spaces.txt -provider digitalocean -enumerate -threads 8
```

## Features

### Enhanced Wordlist Generation
- Base wordlist with common space names
- Company-specific variations
- Custom word integration
- Automatic deduplication

### Intelligent Filtering
- Identifies publicly accessible spaces
- Highlights spaces with interesting permissions
- Filters out non-existent spaces

### Comprehensive Reporting
- Detailed findings summary
- Permission analysis
- Object enumeration results
- Export to file capability

## DigitalOcean Regions Supported

The scanner automatically checks all DigitalOcean regions:
- `nyc3` (New York)
- `sgp1` (Singapore)
- `sfo2` (San Francisco)
- `sfo3` (San Francisco)
- `ams3` (Amsterdam)
- `fra1` (Frankfurt)

## Legal and Ethical Guidelines

### ⚠️ IMPORTANT LEGAL NOTICE

**Only use this tool on authorized targets:**

1. **Bug Bounty Programs Only**: Only scan spaces belonging to companies with active bug bounty programs that explicitly allow this type of testing.

2. **Read the Scope**: Always check the program's scope and rules of engagement before scanning.

3. **Responsible Disclosure**: Report findings through proper channels (HackerOne, Bugcrowd, etc.).

4. **Data Handling**:
   - Don't download sensitive data unnecessarily
   - Don't modify or delete existing data
   - Delete any test files you upload
   - Don't share or distribute found data

5. **Rate Limiting**: Use reasonable thread counts to avoid overwhelming target infrastructure.

### Safe Harbor
Ensure you're covered under the bug bounty program's safe harbor provisions before testing.

## Example Output

```
╔════════════════════════════════════════════════╗
║        DigitalOcean Spaces Scanner             ║
║        Bug Bounty Research Tool                ║
║        Based on S3Scanner                      ║
╚════════════════════════════════════════════════╝

Starting DigitalOcean Spaces scan...
Generated wordlist with 45 entries
Running: s3scanner -bucket-file do_spaces_wordlist.txt -provider digitalocean -threads 4 -json

Scan completed!
Total results: 45
Interesting findings: 2

=== INTERESTING FINDINGS ===
============================================================
DigitalOcean Spaces Security Scan Report
============================================================
Scan completed at: 2024-01-15 10:30:45
Total spaces found: 2

Space Name: company-uploads
Region: nyc3
URL: https://company-uploads.nyc3.digitaloceanspaces.com/
Permissions:
  - Read: ✓
Objects found: 15
  - config.json
  - user_data.csv
  - backup.sql
  ... and 12 more
----------------------------------------

Space Name: assets-staging
Region: sfo3
URL: https://assets-staging.sfo3.digitaloceanspaces.com/
Permissions:
  - Read: ✓
  - Write: ✓
----------------------------------------

✓ Found 2 potentially misconfigured spaces
Remember to:
1. Only test spaces belonging to authorized bug bounty targets
2. Follow responsible disclosure practices
3. Don't download or modify sensitive data
```

## Troubleshooting

### S3Scanner Not Found
```bash
# Check if s3scanner is in PATH
which s3scanner

# If not found, make sure it's installed and in PATH
export PATH=$PATH:/path/to/s3scanner
```

### Permission Denied
```bash
# Make sure the script is executable
chmod +x do-space-scanner.py
```

### No Results
- Check if the wordlist contains relevant terms for your target
- Try increasing the number of threads
- Verify that S3Scanner is working: `s3scanner -version`

## Contributing

This tool is built on top of the excellent [S3Scanner](https://github.com/sa7mon/S3Scanner) by sa7mon. 

For issues related to:
- Core scanning functionality: Report to [S3Scanner repository](https://github.com/sa7mon/S3Scanner)
- DigitalOcean-specific features: Create an issue in this repository

## License

This tool is provided for educational and authorized security research purposes only. Users are responsible for complying with all applicable laws and regulations.

## Disclaimer

This tool is for authorized security testing only. The authors are not responsible for any misuse or damage caused by this tool. Always obtain proper authorization before testing any systems you do not own.