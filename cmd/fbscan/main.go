package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	fb "grayhat-scraper/fbcheck"
)

func main() {
	var apiKey, project, rtdb, bucket, plist, out string
	flag.StringVar(&apiKey, "fb-api-key", "", "Firebase API key")
	flag.StringVar(&project, "fb-project-id", "", "Firebase project id")
	flag.StringVar(&rtdb, "fb-rtdb-url", "", "Firebase Realtime Database URL")
	flag.StringVar(&bucket, "fb-storage-bucket", "", "Firebase Storage bucket (e.g. myapp.appspot.com)")
	flag.StringVar(&plist, "fb-plist", "", "Path to GoogleService-Info.plist/Info.plist to parse")
	flag.StringVar(&out, "out", fmt.Sprintf("fb_report_%d.json", time.Now().Unix()), "Output report file")
	flag.Parse()

	cfg := fb.Config{APIKey: apiKey, ProjectID: project, RTDBURL: rtdb, StorageBucket: bucket, FirestoreProj: project}
	if plist != "" {
		f, err := os.Open(plist)
		if err != nil {
			fmt.Fprintf(os.Stderr, "plist open error: %v\n", err)
		} else {
			defer f.Close()
			pcfg, err := fb.ParsePlist(f)
			if err == nil {
				if cfg.APIKey == "" {
					cfg.APIKey = pcfg.APIKey
				}
				if cfg.ProjectID == "" {
					cfg.ProjectID = pcfg.ProjectID
				}
				if cfg.FirestoreProj == "" {
					cfg.FirestoreProj = pcfg.ProjectID
				}
				if cfg.RTDBURL == "" {
					cfg.RTDBURL = pcfg.RTDBURL
				}
				if cfg.StorageBucket == "" {
					cfg.StorageBucket = pcfg.StorageBucket
				}
			}
		}
	}
	// normalize bucket name
	cfg.StorageBucket = strings.TrimSpace(cfg.StorageBucket)

	rep, err := fb.Run(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scan error: %v\n", err)
		os.Exit(1)
	}
	of, err := os.Create(out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "write error: %v\n", err)
		os.Exit(1)
	}
	defer of.Close()
	enc := json.NewEncoder(of)
	enc.SetIndent("", "  ")
	_ = enc.Encode(rep)
	fmt.Println("Report written:", out)
}
