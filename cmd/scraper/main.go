// grayhat_scraper.go
// High-performance Go port of the asyncio GrayHat Warfare scraper

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gdamore/tcell/v2"
	jsoniter "github.com/json-iterator/go"
	"github.com/rivo/tview"
	"github.com/schollz/progressbar/v3"
)

type Bucket struct {
	URL       string `json:"full_bucket_url"`
	FileCount int    `json:"file_count"`
	ScrapedAt string `json:"scraped_at"`
}

// Structure for files with metadata wrapper
type BucketCollection struct {
	LastUpdated  string   `json:"last_updated,omitempty"`
	TotalBuckets int      `json:"total_buckets,omitempty"`
	Description  string   `json:"description,omitempty"`
	Buckets      []Bucket `json:"buckets"`
}

var (
	outputFile      string
	concurrency     int
	rateLimit       int
	totalRequests   int
	successCount    int
	bucketsFound    int
	newBucketsFound int
	bucketsMutex    sync.Mutex
	buckets         []Bucket
	existingBuckets map[string]bool // Track existing buckets for deduplication
	bar             *progressbar.ProgressBar
	barMutex        sync.Mutex
	jsonLib         = jsoniter.ConfigFastest
	guiMode         bool
	guiUpdates      chan Bucket
	// Config
	sourceURL   string
	minFiles    int
	configMutex sync.RWMutex
	// Diagnostics
	lastStatusCode int
	errorCount     int
)

func init() {
	flag.StringVar(&outputFile, "o", "merged_deduplicated.json", "Output JSON file (default: merged_deduplicated.json)")
	flag.IntVar(&concurrency, "c", 200, "Max concurrent workers")
	flag.IntVar(&rateLimit, "r", 100, "Requests per second limit")
	flag.BoolVar(&guiMode, "gui", false, "Run with terminal UI that shows buckets in real-time (disabled by default)")
	flag.StringVar(&sourceURL, "url", "https://buckets.grayhatwarfare.com/random/buckets", "Source page URL to scrape")
	flag.IntVar(&minFiles, "min", 1000, "Minimum file count to accept a bucket")
}

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())

	// Load existing buckets for deduplication
	fmt.Println("🔍 Loading existing buckets for deduplication...")
	loadExistingBuckets()
	fmt.Printf("📊 Loaded %d existing buckets\n", len(existingBuckets))

	// Handle SIGINT/SIGTERM
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		fmt.Println("\n🛑 Interrupt received, stopping...")
		cancel()
	}()

	// GUI mode removed: always run in CLI mode
	guiMode = false

	// CLI mode (no GUI)
	bar = progressbar.NewOptions(-1,
		progressbar.OptionSetDescription("Pages: 0 | New: 0 | Total: 0"),
		progressbar.OptionShowCount(),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionThrottle(50*time.Millisecond),
		progressbar.OptionSetWidth(50),
	)
	runScraper(ctx)
}

func scrape() {
	configMutex.RLock()
	currentURL := sourceURL
	configMutex.RUnlock()
	req, _ := http.NewRequest("GET", currentURL, nil)
	req.Header.Set("User-Agent", randomUserAgent())
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	totalRequests++
	if err != nil || resp.StatusCode != 200 {
		if err != nil {
			errorCount++
		}
		if resp != nil {
			lastStatusCode = resp.StatusCode
			resp.Body.Close()
		}
		return
	}
	lastStatusCode = resp.StatusCode
	successCount++
	defer resp.Body.Close()
	processResponse(resp.Body)
}

func processResponse(body io.Reader) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return
	}
	doc.Find("table.table-bordered tbody tr").Each(func(i int, s *goquery.Selection) {
		cols := s.Find("td")
		if cols.Length() < 4 {
			return
		}

		// Extract bucket URL from href
		bucketURL, exists := cols.Eq(1).Find("a").Attr("href")
		if !exists || !strings.Contains(bucketURL, "bucket=") {
			return
		}

		// Properly extract and decode bucket parameter
		bucketParam := extractBucketParam(bucketURL)
		if bucketParam == "" {
			return
		}

		decoded, err := url.QueryUnescape(bucketParam)
		if err != nil {
			decoded = bucketParam // fallback to original if decode fails
		}

		// Check for target keywords (expanded list)
		targetKeywords := []string{""}
		lowerBucket := strings.ToLower(decoded)
		found := false
		for _, keyword := range targetKeywords {
			if strings.Contains(lowerBucket, keyword) {
				found = true
				break
			}
		}
		if !found {
			return
		}

		// Extract file count
		countText := strings.TrimSpace(cols.Eq(2).Text())
		count := parseFileCount(countText)
		configMutex.RLock()
		min := minFiles
		configMutex.RUnlock()
		if count < min {
			return
		}

		// Check if this bucket already exists
		bucketsMutex.Lock()
		if existingBuckets[decoded] {
			// Duplicate found, skip
			bucketsMutex.Unlock()
			return
		}

		// New bucket found - add to both collections
		b := Bucket{
			URL:       decoded,
			FileCount: count,
			ScrapedAt: time.Now().Format(time.RFC3339),
		}
		buckets = append(buckets, b)
		existingBuckets[decoded] = true
		bucketsFound++
		newBucketsFound++
		// Notify GUI if enabled
		if guiUpdates != nil {
			select {
			case guiUpdates <- b:
			default:
			}
		}
		bucketsMutex.Unlock()
	})
}

func updateProgressBar() {
	if guiMode {
		// In GUI mode we do not draw the CLI progress bar
		return
	}
	barMutex.Lock()
	defer barMutex.Unlock()
	description := fmt.Sprintf("Pages: %d | New: %d | Total: %d", totalRequests, newBucketsFound, bucketsFound)
	bar.Describe(description)
	bar.Add(1)
}

func loadExistingBuckets() {
	existingBuckets = make(map[string]bool)

	// Try to load existing file
	file, err := os.Open(outputFile)
	if err != nil {
		// File doesn't exist yet, start fresh
		return
	}
	defer file.Close()

	// Try to parse as BucketCollection first
	var collection BucketCollection
	decoder := jsonLib.NewDecoder(file)
	if err := decoder.Decode(&collection); err == nil && len(collection.Buckets) > 0 {
		// Load existing buckets into our collections
		for _, bucket := range collection.Buckets {
			existingBuckets[bucket.URL] = true
			buckets = append(buckets, bucket)
		}
		bucketsFound = len(collection.Buckets)
		return
	}

	// Try as simple array
	file.Seek(0, 0)
	decoder = jsonLib.NewDecoder(file)
	var simpleBuckets []Bucket
	if err := decoder.Decode(&simpleBuckets); err == nil {
		for _, bucket := range simpleBuckets {
			existingBuckets[bucket.URL] = true
			buckets = append(buckets, bucket)
		}
		bucketsFound = len(simpleBuckets)
	}
}

func saveResults() {
	if newBucketsFound == 0 {
		fmt.Printf("💾 No new buckets to save\n")
		return
	}

	// Create the merged collection with metadata
	collection := BucketCollection{
		LastUpdated:  time.Now().Format(time.RFC3339),
		TotalBuckets: len(buckets),
		Description:  "Merged and deduplicated buckets from scraping sessions",
		Buckets:      buckets,
	}

	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Printf("❌ Error creating output file: %v\n", err)
		return
	}
	defer file.Close()

	encoder := jsonLib.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.Encode(collection)
}

func extractBucketParam(bucketURL string) string {
	// Find bucket= parameter
	start := strings.Index(bucketURL, "bucket=")
	if start == -1 {
		return ""
	}
	start += 7 // length of "bucket="

	// Find end of parameter (& or end of string)
	end := strings.Index(bucketURL[start:], "&")
	if end == -1 {
		return bucketURL[start:]
	}
	return bucketURL[start : start+end]
}

func parseFileCount(s string) int {
	// Remove commas and whitespace
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)

	// Try to parse as integer
	if count, err := strconv.Atoi(s); err == nil {
		return count
	}

	// Fallback: try to extract number from text
	var count int
	fmt.Sscanf(s, "%d", &count)
	return count
}

// Removed urlDecode function - now using proper url.QueryUnescape

func randomUserAgent() string {
	agents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 Version/17.1 Safari/605.1.15",
	}
	return agents[rand.Intn(len(agents))]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// runScraper contains the main scraping loop. It can run in CLI or GUI mode.
func runScraper(ctx context.Context) {
	ticker := time.NewTicker(time.Second / time.Duration(rateLimit))
	defer ticker.Stop()
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			saveResults()
			fmt.Printf("\n🏁 Done! Total: %d | New Found: %d | Success rate: %.1f%%\n",
				totalRequests, newBucketsFound, float64(successCount)/float64(max(totalRequests, 1))*100)
			fmt.Printf("💾 Added %d new buckets to %s\n", newBucketsFound, outputFile)
			return
		case <-ticker.C:
			sem <- struct{}{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				scrape()
				<-sem
				updateProgressBar()
			}()
		}
	}
}

// startGUI sets up a terminal UI and streams discovered buckets in real-time.
func startGUI(ctx context.Context, cancel context.CancelFunc) {
	app := tview.NewApplication()

	// List to show buckets
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(true).SetTitle("S3eker - Discovered Buckets")

	// Status text
	status := tview.NewTextView().SetDynamicColors(true)
	status.SetBorder(true).SetTitle("Status")
	updateStatus := func() {
		configMutex.RLock()
		u := sourceURL
		min := minFiles
		configMutex.RUnlock()
		host := u
		if parsed, err := url.Parse(u); err == nil {
			host = parsed.Host
		}
		status.Clear()
		fmt.Fprintf(status, "URL: %s\nPages: %d  New: %d  Total: %d  HTTP: %d  Errors: %d  Min: %d\n[::b]Keys[::-] q=quit  u=change URL  m=min files\n", host, totalRequests, newBucketsFound, bucketsFound, lastStatusCode, errorCount, min)
	}
	status.SetText("Initializing...")

	// Layout
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(list, 0, 1, true).
		AddItem(status, 3, 0, false)

	pages := tview.NewPages()
	pages.AddPage("main", flex, true, true)

	// Keybindings
	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC:
			cancel()
			app.Stop()
			return nil
		}
		switch event.Rune() {
		case 'q', 'Q':
			cancel()
			app.Stop()
			return nil
		case 'u', 'U':
			// Show URL input form
			configMutex.RLock()
			current := sourceURL
			configMutex.RUnlock()
			form := tview.NewForm()
			form.AddInputField("Source URL", current, 100, nil, nil)
			form.AddButton("Save", func() {
				ifItem := form.GetFormItemByLabel("Source URL")
				if input, ok := ifItem.(*tview.InputField); ok {
					text := strings.TrimSpace(input.GetText())
					if text != "" {
						configMutex.Lock()
						sourceURL = text
						configMutex.Unlock()
						updateStatus()
					}
				}
				pages.RemovePage("url")
			})
			form.AddButton("Cancel", func() { pages.RemovePage("url") })
			form.SetBorder(true).SetTitle("Set Source URL")
			modal := tview.NewFlex().
				AddItem(nil, 0, 1, false).
				AddItem(form, 10, 0, true).
				AddItem(nil, 0, 1, false)
			pages.AddPage("url", modal, true, true)
			app.SetFocus(form)
			return nil
		case 'm', 'M':
			// Change minimum files threshold
			configMutex.RLock()
			currentMin := minFiles
			configMutex.RUnlock()
			form := tview.NewForm()
			form.AddInputField("Min Files", fmt.Sprintf("%d", currentMin), 10, nil, nil)
			form.AddButton("Save", func() {
				ifItem := form.GetFormItemByLabel("Min Files")
				if input, ok := ifItem.(*tview.InputField); ok {
					text := strings.TrimSpace(input.GetText())
					if v, err := strconv.Atoi(text); err == nil && v >= 0 {
						configMutex.Lock()
						minFiles = v
						configMutex.Unlock()
						updateStatus()
					}
				}
				pages.RemovePage("min")
			})
			form.AddButton("Cancel", func() { pages.RemovePage("min") })
			form.SetBorder(true).SetTitle("Set Minimum Files")
			modal := tview.NewFlex().
				AddItem(nil, 0, 1, false).
				AddItem(form, 10, 0, true).
				AddItem(nil, 0, 1, false)
			pages.AddPage("min", modal, true, true)
			app.SetFocus(form)
			return nil
		}
		return event
	})

	// Preload existing buckets (if any)
	bucketsMutex.Lock()
	for _, b := range buckets {
		list.AddItem(b.URL, "", 0, nil)
	}
	bucketsMutex.Unlock()
	updateStatus()

	// Start scraper in background
	go runScraper(ctx)

	// Periodic status refresh in GUI
	go func() {
		t := time.NewTicker(1 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				_ = app.QueueUpdateDraw(func() { updateStatus() })
			}
		}
	}()

	// Consume GUI updates
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case b := <-guiUpdates:
				_ = app.QueueUpdateDraw(func() {
					list.AddItem(fmt.Sprintf("%s  (%d files)", b.URL, b.FileCount), "", 0, nil)
					status.Clear()
					updateStatus()
				})
			}
		}
	}()

	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "UI error: %v\n", err)
		cancel()
	}
}
