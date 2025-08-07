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
	"github.com/schollz/progressbar/v3"
	"github.com/json-iterator/go"
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
	outputFile    string
	concurrency   int
	rateLimit     int
	totalRequests int
	successCount  int
	bucketsFound  int
	newBucketsFound int
	bucketsMutex  sync.Mutex
	buckets       []Bucket
	existingBuckets map[string]bool  // Track existing buckets for deduplication
	bar          *progressbar.ProgressBar
	barMutex     sync.Mutex
	jsonLib      = jsoniter.ConfigFastest
)

func init() {
	flag.StringVar(&outputFile, "o", "merged_deduplicated.json", "Output JSON file (default: merged_deduplicated.json)")
	flag.IntVar(&concurrency, "c", 200, "Max concurrent workers")
	flag.IntVar(&rateLimit, "r", 100, "Requests per second limit")
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

	bar = progressbar.NewOptions(-1,
		progressbar.OptionSetDescription("Pages: 0 | New: 0 | Total: 0"),
		progressbar.OptionShowCount(),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionThrottle(50*time.Millisecond),
		progressbar.OptionSetWidth(50),
	)

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

func scrape() {
	url := "https://buckets.grayhatwarfare.com/random/buckets"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", randomUserAgent())
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	totalRequests++
	if err != nil || resp.StatusCode != 200 {
		return
	}
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
		if count < 1000 {
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
		buckets = append(buckets, Bucket{
			URL:       decoded,
			FileCount: count,
			ScrapedAt: time.Now().Format(time.RFC3339),
		})
		existingBuckets[decoded] = true
		bucketsFound++
		newBucketsFound++
		bucketsMutex.Unlock()
	})
}

func updateProgressBar() {
	barMutex.Lock()
	defer barMutex.Unlock()
	
	// Update progress bar description with pages, new buckets, and total buckets
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