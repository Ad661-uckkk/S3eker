package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	fb "grayhat-scraper/fbcheck"

	"golang.org/x/term"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	configureColors()
	printHeader()
	fmt.Println("Select a mode:")
	fmt.Println("  1) Scrape Grayhat for new open buckets (CLI)")
	fmt.Println("  2) Check Firebase configuration (wizard)")
	fmt.Println("  3) Close")
	fmt.Print(colorCyan + "> " + colorReset)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	switch choice {
	case "1":
		runScraperCLI()
		return
	case "2":
		runFirebaseWizard(reader)
		return
	case "3":
		fmt.Println("Goodbye.")
		return
	default:
		fmt.Println("Invalid choice. Exiting.")
		os.Exit(1)
	}
}

func runScraperCLI() {
	// Run the current repository's scraper in CLI mode (no TUI)
	exe, err := os.Executable()
	if err != nil {
		fmt.Println("Could not resolve current executable:", err)
	}
	_ = exe // not used but kept for clarity
	// Prefer running local 'main.go' built binary if exists
	// Fallback to calling itself with flags if packaging changes in future.

	// If the scraper is embedded as the main in repo root, just exec with -gui=false
	// Attempt to find current binary path name and re-invoke with -gui=false if needed.
	// Otherwise look for known names in PATH.
	candidates := []string{"s3eker-scraper", "grayhat-scraper", "s3eker"}
	var bin string
	var exeInfo os.FileInfo
	if exe != "" {
		if fi, e := os.Stat(exe); e == nil {
			exeInfo = fi
		}
	}
	for _, c := range candidates {
		if p, err := exec.LookPath(c); err == nil {
			// avoid recursion: skip if resolves to current executable
			if exeInfo != nil {
				if fi, e2 := os.Stat(p); e2 == nil && os.SameFile(exeInfo, fi) {
					continue
				}
			}
			bin = p
			break
		}
	}
	if bin == "" {
		// Try sibling binary in the same directory as the current executable
		if exe != "" {
			cand := filepath.Join(filepath.Dir(exe), "s3eker-scraper")
			if fi, e := os.Stat(cand); e == nil && !fi.IsDir() {
				bin = cand
			}
		}
	}
	if bin == "" {
		fmt.Println("Could not find scraper binary. Build it first: 'go build -o s3eker-scraper ./'.")
		os.Exit(1)
	}
	cmd := exec.Command(bin, "-gui=false")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Println("Scraper exited:", err)
	}
}

func prompt(reader *bufio.Reader, label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	s, _ := reader.ReadString('\n')
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	return s
}

func runFirebaseWizard(reader *bufio.Reader) {
	fmt.Println(colorBold + "Firebase checker" + colorReset + " – provide a GoogleService-Info.plist OR answer prompts.")
	usePlist := strings.ToLower(prompt(reader, "Use plist? (y/N)", "n"))

	cfg := fb.Config{}
	if usePlist == "y" || usePlist == "yes" {
		plistPath := prompt(reader, "Path to GoogleService-Info.plist", "")
		f, err := os.Open(plistPath)
		if err != nil {
			fmt.Println("Error opening plist:", err)
			os.Exit(1)
		}
		defer f.Close()
		pcfg, err := fb.ParsePlist(f)
		if err != nil {
			fmt.Println("Error parsing plist:", err)
			os.Exit(1)
		}
		cfg = pcfg
	} else {
		cfg.APIKey = prompt(reader, "Firebase API key", "")
		cfg.ProjectID = prompt(reader, "Firebase project id", "")
		cfg.RTDBURL = prompt(reader, "Firebase Realtime Database URL", "")
		cfg.StorageBucket = prompt(reader, "Firebase Storage bucket (e.g. myapp.appspot.com)", "")
		if cfg.FirestoreProj == "" {
			cfg.FirestoreProj = cfg.ProjectID
		}
	}

	out := prompt(reader, "Output file", fmt.Sprintf("fb_report_%d.json", time.Now().Unix()))

	rep, err := fb.Run(cfg)
	if err != nil {
		fmt.Println("Scan error:", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil && filepath.Dir(out) != "." {
		fmt.Println("Warning: could not create directory:", err)
	}
	f, err := os.Create(out)
	if err != nil {
		fmt.Println("Write error:", err)
		os.Exit(1)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	_ = enc.Encode(rep)

	fmt.Println()
	fmt.Println(colorBold+"Report written:"+colorReset, out)
	// Print formatted summary
	fmt.Println()
	fmt.Println(colorBold + "Findings" + colorReset)
	width := termWidth()
	printFindingsTable(rep.Findings, width)
	// Totals
	pass, warn, fail, info := 0, 0, 0, 0
	for _, fi := range rep.Findings {
		switch fi.Status {
		case "PASS":
			pass++
		case "WARN":
			warn++
		case "FAIL":
			fail++
		default:
			info++
		}
	}
	fmt.Printf("Totals: %s%d PASS%s, %s%d WARN%s, %s%d FAIL%s, %d INFO\n", colorGreen, pass, colorReset, colorYellow, warn, colorReset, colorRed, fail, colorReset, info)

	// Recommendations for any non-PASS
	recs := recommendations(rep.Findings)
	if len(recs) > 0 {
		fmt.Println()
		fmt.Println(colorBold + "Recommendations" + colorReset)
		for _, r := range recs {
			for _, line := range wrapText("- "+r, width) {
				fmt.Println(line)
			}
		}
	}

	// Optional export as Markdown
	exp := strings.ToLower(prompt(reader, "Export summary as Markdown? (y/N)", "n"))
	if exp == "y" || exp == "yes" {
		md := toMarkdown(rep.Findings, pass, warn, fail, info)
		mdPath := strings.TrimSuffix(out, filepath.Ext(out)) + ".md"
		if err := os.WriteFile(mdPath, []byte(md), 0o644); err == nil {
			fmt.Println(colorBold+"Markdown saved:"+colorReset, mdPath)
		}
	}
}

// ───────────────────────── UI helpers ─────────────────────────
var (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorCyan   = "\033[36m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
)

var useColor = true

func configureColors() {
	if os.Getenv("NO_COLOR") != "" || !term.IsTerminal(int(os.Stdout.Fd())) {
		useColor = false
	}
	if !useColor {
		// disable colors
		resetAllColors()
	}
}

func printHeader() {
	ascii := []string{
		" ░▒▓███████▓▒░ ░▒▓███████▓▒░  ░▒▓████████▓▒░ ░▒▓█▓▒░░▒▓█▓▒░ ░▒▓████████▓▒░ ░▒▓███████▓▒░  ",
		"░▒▓█▓▒░               ░▒▓█▓▒░ ░▒▓█▓▒░        ░▒▓█▓▒░░▒▓█▓▒░ ░▒▓█▓▒░        ░▒▓█▓▒░░▒▓█▓▒░ ",
		"░▒▓█▓▒░               ░▒▓█▓▒░ ░▒▓█▓▒░        ░▒▓█▓▒░░▒▓█▓▒░ ░▒▓█▓▒░        ░▒▓█▓▒░░▒▓█▓▒░ ",
		" ░▒▓██████▓▒░  ░▒▓███████▓▒░  ░▒▓██████▓▒░   ░▒▓███████▓▒░  ░▒▓██████▓▒░   ░▒▓███████▓▒░  ",
		"       ░▒▓█▓▒░        ░▒▓█▓▒░ ░▒▓█▓▒░        ░▒▓█▓▒░░▒▓█▓▒░ ░▒▓█▓▒░        ░▒▓█▓▒░░▒▓█▓▒░ ",
		"       ░▒▓█▓▒░        ░▒▓█▓▒░ ░▒▓█▓▒░        ░▒▓█▓▒░░▒▓█▓▒░ ░▒▓█▓▒░        ░▒▓█▓▒░░▒▓█▓▒░ ",
		"░▒▓███████▓▒░  ░▒▓███████▓▒░  ░▒▓████████▓▒░ ░▒▓█▓▒░░▒▓█▓▒░ ░▒▓████████▓▒░ ░▒▓█▓▒░░▒▓█▓▒░ ",
		"                                                                                          ",
		"                                                                                          ",
	}
	fmt.Println(colorCyan)
	for _, line := range ascii {
		fmt.Println(line)
	}
	fmt.Println(colorReset)
	fmt.Println("License: MIT · (c) Contributors. Use ethically and with authorization.")
	fmt.Println("Tips: reports are JSON; scraper runs in CLI; Firebase checker is read-only by default.")
	fmt.Println()
}

func statusColor(s string) string {
	switch s {
	case "PASS":
		return colorGreen
	case "WARN":
		return colorYellow
	case "FAIL":
		return colorRed
	default:
		return colorCyan
	}
}

func resetAllColors() {
	colorReset = ""
	colorBold = ""
	colorCyan = ""
	colorGreen = ""
	colorYellow = ""
	colorRed = ""
}

func termWidth() int {
	w := 80
	if term.IsTerminal(int(os.Stdout.Fd())) {
		if cols, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && cols > 0 {
			w = cols
		}
	}
	return w
}

func wrapText(s string, width int) []string {
	if width <= 20 {
		width = 80
	}
	max := width - 4
	var out []string
	for len(s) > max {
		cut := max
		// try to break at space
		for i := max; i > max-20 && i > 0; i-- {
			if s[i] == ' ' {
				cut = i
				break
			}
		}
		out = append(out, s[:cut])
		s = strings.TrimSpace(s[cut:])
	}
	if s != "" {
		out = append(out, s)
	}
	return out
}

func printFindingsTable(rows []fb.Finding, width int) {
	nameW := 22
	divider := strings.Repeat("─", min(width, 60))
	fmt.Println(divider)
	for _, fi := range rows {
		stColor := statusColor(fi.Status)
		name := fi.Name
		if len(name) > nameW {
			name = name[:nameW]
		}
		first := true
		for _, line := range wrapText(fi.Detail, width-nameW-12) {
			if first {
				fmt.Printf("%-*s  %s%-5s%s  %s\n", nameW, name, stColor, fi.Status, colorReset, line)
				first = false
			} else {
				fmt.Printf("%-*s  %-5s  %s\n", nameW, "", "", line)
			}
		}
	}
	fmt.Println(divider)
}

func recommendations(rows []fb.Finding) []string {
	var out []string
	for _, fi := range rows {
		if fi.Status == "FAIL" || fi.Status == "WARN" {
			switch fi.Name {
			case "RTDBPublicRead":
				out = append(out, "Realtime Database: set rules to deny unauthenticated reads ('.read': false) or restrict by auth condition.")
			case "StoragePublicList":
				out = append(out, "Storage: disable public listing; use security rules to require auth and least‑privilege access.")
			case "StorageWrite":
				out = append(out, "Storage: block unauthenticated writes; restrict write paths to authenticated users only.")
			case "AnonymousAuth":
				out = append(out, "Auth: disable anonymous authentication if not needed.")
			case "SignUp":
				out = append(out, "Auth: disable public sign‑up or enforce allow‑list if this backend is not intended for public accounts.")
			}
		}
	}
	return out
}

func toMarkdown(rows []fb.Finding, pass, warn, fail, info int) string {
	b := &strings.Builder{}
	fmt.Fprintln(b, "# S3eker Firebase Report")
	fmt.Fprintf(b, "\nTotals: %d PASS, %d WARN, %d FAIL, %d INFO\n\n", pass, warn, fail, info)
	fmt.Fprintln(b, "| Check | Status | Detail |")
	fmt.Fprintln(b, "|---|---|---|")
	for _, fi := range rows {
		fmt.Fprintf(b, "| %s | %s | %s |\n", fi.Name, fi.Status, strings.ReplaceAll(fi.Detail, "|", "/"))
	}
	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
