package fbcheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	plist "howett.net/plist"
)

type Config struct {
	APIKey        string
	ProjectID     string
	RTDBURL       string
	StorageBucket string
	FirestoreProj string
	PlistPath     string
}

type Finding struct {
	Name   string `json:"name"`
	Status string `json:"status"` // PASS/WARN/FAIL/INFO
	Detail string `json:"detail"`
}

type Report struct {
	Timestamp time.Time `json:"timestamp"`
	Config    Config    `json:"config"`
	Findings  []Finding `json:"findings"`
}

func add(findings *[]Finding, name, status, detail string) {
	*findings = append(*findings, Finding{Name: name, Status: status, Detail: detail})
}

// ParsePlist extracts Firebase settings from a GoogleService-Info.plist/Info.plist file.
func ParsePlist(r io.Reader) (cfg Config, _ error) {
	var data map[string]any
	// read all since decoder expects ReadSeeker
	buf, err := io.ReadAll(r)
	if err != nil {
		return cfg, err
	}
	dec := plist.NewDecoder(bytes.NewReader(buf))
	if err := dec.Decode(&data); err != nil {
		return cfg, err
	}
	if v, ok := data["API_KEY"].(string); ok {
		cfg.APIKey = v
	}
	if v, ok := data["PROJECT_ID"].(string); ok {
		cfg.ProjectID = v
	}
	if v, ok := data["DATABASE_URL"].(string); ok {
		cfg.RTDBURL = v
	}
	if v, ok := data["STORAGE_BUCKET"].(string); ok {
		cfg.StorageBucket = v
	}
	if cfg.FirestoreProj == "" {
		cfg.FirestoreProj = cfg.ProjectID
	}
	return cfg, nil
}

func httpClient() *http.Client { return &http.Client{Timeout: 15 * time.Second} }

func signInAnonymously(apiKey string) (string, int, error) {
	if apiKey == "" {
		return "", 0, nil
	}
	endpoint := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signInAnonymously?key=%s", apiKey)
	body := bytes.NewBufferString(`{"returnSecureToken":true}`)
	resp, err := httpClient().Post(endpoint, "application/json", body)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	var m map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&m)
	token, _ := m["idToken"].(string)
	return token, resp.StatusCode, nil
}

func trySignUp(apiKey string) (bool, int, error) {
	if apiKey == "" {
		return false, 0, nil
	}
	endpoint := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signUp?key=%s", apiKey)
	// RFC 2606 reserved domain to avoid sending to real domains
	email := fmt.Sprintf("s3eker+%d@example.test", time.Now().Unix())
	payload := fmt.Sprintf(`{"email":"%s","password":"s3ekerTest123!","returnSecureToken":true}`, email)
	resp, err := httpClient().Post(endpoint, "application/json", strings.NewReader(payload))
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return true, resp.StatusCode, nil
	}
	return false, resp.StatusCode, nil
}

func checkRTDB(rtdbURL, idToken string) (publicReadable, anonReadable bool, statusPublic, statusAnon int, err error) {
	if rtdbURL == "" {
		return
	}
	// ensure URL ends with /.json
	base := strings.TrimRight(rtdbURL, "/") + "/.json"
	// No-auth
	resp, e := httpClient().Get(base)
	if e != nil {
		err = e
		return
	}
	statusPublic = resp.StatusCode
	resp.Body.Close()
	if statusPublic == http.StatusOK {
		publicReadable = true
	}
	// With anon token if present
	if idToken != "" {
		u := base + "?auth=" + url.QueryEscape(idToken)
		r2, e2 := httpClient().Get(u)
		if e2 != nil {
			err = e2
			return
		}
		statusAnon = r2.StatusCode
		r2.Body.Close()
		if statusAnon == http.StatusOK {
			anonReadable = true
		}
	}
	return
}

func checkStorageList(bucket, idToken string) (publicList, anonList bool, statusPublic, statusAnon int, err error) {
	if bucket == "" {
		return
	}
	base := fmt.Sprintf("https://firebasestorage.googleapis.com/v0/b/%s/o", bucket)
	// No-auth
	resp, e := httpClient().Get(base)
	if e != nil {
		err = e
		return
	}
	statusPublic = resp.StatusCode
	resp.Body.Close()
	if statusPublic == http.StatusOK {
		publicList = true
	}
	if idToken != "" {
		req, _ := http.NewRequest("GET", base, nil)
		req.Header.Set("Authorization", "Bearer "+idToken)
		r2, e2 := httpClient().Do(req)
		if e2 != nil {
			err = e2
			return
		}
		statusAnon = r2.StatusCode
		r2.Body.Close()
		if statusAnon == http.StatusOK {
			anonList = true
		}
	}
	return
}

func tryStorageWriteDelete(bucket, idToken string) (writeOK, deleteOK bool, statusWrite, statusDelete int, err error) {
	if bucket == "" {
		return
	}
	name := fmt.Sprintf("s3eker_probe/%d.txt", time.Now().UnixNano())
	uploadURL := fmt.Sprintf("https://firebasestorage.googleapis.com/v0/b/%s/o?uploadType=media&name=%s", bucket, url.QueryEscape(name))
	req, _ := http.NewRequest("POST", uploadURL, strings.NewReader("s3eker probe"))
	req.Header.Set("Content-Type", "text/plain")
	if idToken != "" {
		req.Header.Set("Authorization", "Bearer "+idToken)
	}
	resp, e := httpClient().Do(req)
	if e != nil {
		err = e
		return
	}
	statusWrite = resp.StatusCode
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if statusWrite == http.StatusOK {
		writeOK = true
	}

	// Attempt delete of the object we just created
	delURL := fmt.Sprintf("https://firebasestorage.googleapis.com/v0/b/%s/o/%s", bucket, url.PathEscape(name))
	dreq, _ := http.NewRequest("DELETE", delURL, nil)
	if idToken != "" {
		dreq.Header.Set("Authorization", "Bearer "+idToken)
	}
	dresp, e2 := httpClient().Do(dreq)
	if e2 != nil {
		err = e2
		return
	}
	statusDelete = dresp.StatusCode
	io.Copy(io.Discard, dresp.Body)
	dresp.Body.Close()
	if statusDelete == http.StatusOK {
		deleteOK = true
	}
	return
}

func checkFirestoreList(project, idToken string) (publicList, anonList bool, statusPublic, statusAnon int, err error) {
	if project == "" {
		return
	}
	base := fmt.Sprintf("https://firestore.googleapis.com/v1/projects/%s/databases/(default)/documents", project)
	// No-auth
	resp, e := httpClient().Get(base)
	if e != nil {
		err = e
		return
	}
	statusPublic = resp.StatusCode
	resp.Body.Close()
	if statusPublic == http.StatusOK {
		publicList = true
	}
	if idToken != "" {
		req, _ := http.NewRequest("GET", base, nil)
		req.Header.Set("Authorization", "Bearer "+idToken)
		r2, e2 := httpClient().Do(req)
		if e2 != nil {
			err = e2
			return
		}
		statusAnon = r2.StatusCode
		r2.Body.Close()
		if statusAnon == http.StatusOK {
			anonList = true
		}
	}
	return
}

// Run performs the scan and returns a report.
func Run(cfg Config) (Report, error) {
	rep := Report{Timestamp: time.Now(), Config: cfg}
	// Anonymous auth
	anonToken, stAnon, err := signInAnonymously(cfg.APIKey)
	if err != nil {
		add(&rep.Findings, "AnonymousAuth", "INFO", fmt.Sprintf("error: %v", err))
	} else {
		if stAnon == http.StatusOK && anonToken != "" {
			add(&rep.Findings, "AnonymousAuth", "FAIL", "Anonymous authentication ENABLED")
		} else {
			add(&rep.Findings, "AnonymousAuth", "PASS", fmt.Sprintf("status=%d", stAnon))
		}
	}

	// Try sign-up (as requested, enabled in normal mode)
	if cfg.APIKey != "" {
		ok, st, err := trySignUp(cfg.APIKey)
		if err != nil {
			add(&rep.Findings, "SignUp", "INFO", fmt.Sprintf("error: %v", err))
		} else if ok {
			add(&rep.Findings, "SignUp", "FAIL", "Unauthenticated sign-up ENABLED (account created & token issued)")
		} else {
			add(&rep.Findings, "SignUp", "PASS", fmt.Sprintf("status=%d", st))
		}
	}

	// RTDB checks
	pub, an, sp, sa, err := checkRTDB(cfg.RTDBURL, anonToken)
	if err != nil {
		add(&rep.Findings, "RTDB", "INFO", fmt.Sprintf("error: %v", err))
	} else {
		if pub {
			add(&rep.Findings, "RTDBPublicRead", "FAIL", fmt.Sprintf("/.json readable without auth (status=%d)", sp))
		} else {
			add(&rep.Findings, "RTDBPublicRead", "PASS", fmt.Sprintf("status=%d", sp))
		}
		if an {
			add(&rep.Findings, "RTDBAnonRead", "WARN", fmt.Sprintf("/.json readable with anonymous token (status=%d)", sa))
		} else {
			add(&rep.Findings, "RTDBAnonRead", "PASS", fmt.Sprintf("status=%d", sa))
		}
	}

	// Storage list + write/delete
	spl, sal, sps, sas, err := checkStorageList(cfg.StorageBucket, anonToken)
	if err != nil {
		add(&rep.Findings, "StorageList", "INFO", fmt.Sprintf("error: %v", err))
	} else {
		if spl {
			add(&rep.Findings, "StoragePublicList", "FAIL", fmt.Sprintf("listable without auth (status=%d)", sps))
		} else {
			add(&rep.Findings, "StoragePublicList", "PASS", fmt.Sprintf("status=%d", sps))
		}
		if sal {
			add(&rep.Findings, "StorageAnonList", "WARN", fmt.Sprintf("listable with anonymous token (status=%d)", sas))
		} else {
			add(&rep.Findings, "StorageAnonList", "PASS", fmt.Sprintf("status=%d", sas))
		}
	}
	wOK, dOK, sw, sd, err := tryStorageWriteDelete(cfg.StorageBucket, anonToken)
	if cfg.StorageBucket != "" {
		if err != nil {
			add(&rep.Findings, "StorageWriteDelete", "INFO", fmt.Sprintf("error: %v", err))
		} else {
			if wOK {
				add(&rep.Findings, "StorageWrite", "FAIL", fmt.Sprintf("write allowed (status=%d)", sw))
			} else {
				add(&rep.Findings, "StorageWrite", "PASS", fmt.Sprintf("status=%d", sw))
			}
			if dOK {
				add(&rep.Findings, "StorageDelete", "FAIL", fmt.Sprintf("delete allowed for probe (status=%d)", sd))
			} else {
				add(&rep.Findings, "StorageDelete", "PASS", fmt.Sprintf("status=%d", sd))
			}
		}
	}

	// Firestore
	fpl, fal, fps, fas, err := checkFirestoreList(cfg.FirestoreProj, anonToken)
	if err != nil {
		add(&rep.Findings, "FirestoreList", "INFO", fmt.Sprintf("error: %v", err))
	} else {
		if fpl {
			add(&rep.Findings, "FirestorePublicRead", "FAIL", fmt.Sprintf("listable without auth (status=%d)", fps))
		} else {
			add(&rep.Findings, "FirestorePublicRead", "PASS", fmt.Sprintf("status=%d", fps))
		}
		if fal {
			add(&rep.Findings, "FirestoreAnonRead", "WARN", fmt.Sprintf("listable with anonymous token (status=%d)", fas))
		} else {
			add(&rep.Findings, "FirestoreAnonRead", "PASS", fmt.Sprintf("status=%d", fas))
		}
	}

	return rep, nil
}
