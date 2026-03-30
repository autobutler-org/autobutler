package main

import (
	"archive/zip"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	provisioningSecret string
	githubToken        string
	rateLimiter        = &ipRateLimiter{hits: make(map[string][]time.Time)}
	artifactCache      = &cache{}
)

const (
	ghAPIBase     = "https://api.github.com"
	repoPath      = "autobutler-org/autobutler"
	artifactName  = "autobutler-linux-arm64"
	cacheTTL      = 60 * time.Second
	rateLimit     = 5
	rateWindow    = time.Hour
	listenAddr    = ":8090"
)

type ipRateLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

func (r *ipRateLimiter) allow(ip string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-rateWindow)
	var recent []time.Time
	for _, t := range r.hits[ip] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	if len(recent) >= rateLimit {
		r.hits[ip] = recent
		return false
	}
	r.hits[ip] = append(recent, now)
	return true
}

type cache struct {
	mu        sync.Mutex
	data      []branchArtifact
	fetchedAt time.Time
}

func (c *cache) get() ([]branchArtifact, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data != nil && time.Since(c.fetchedAt) < cacheTTL {
		return c.data, true
	}
	return nil, false
}

func (c *cache) set(data []branchArtifact) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = data
	c.fetchedAt = time.Now()
}

type branchArtifact struct {
	Branch     string `json:"branch"`
	PRNumber   int    `json:"pr_number,omitempty"`
	PRTitle    string `json:"pr_title,omitempty"`
	BuiltAt    string `json:"built_at"`
	ArtifactID int64  `json:"artifact_id"`
}

type ghArtifactsResponse struct {
	Artifacts []ghArtifact `json:"artifacts"`
}

type ghArtifact struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	WorkflowRunID      int64  `json:"workflow_run_id,omitempty"`
	CreatedAt          string `json:"created_at"`
	Expired            bool   `json:"expired"`
	WorkflowRun        *ghArtifactWorkflowRun `json:"workflow_run,omitempty"`
}

type ghArtifactWorkflowRun struct {
	ID         int64  `json:"id"`
	HeadBranch string `json:"head_branch"`
}

type ghWorkflowRun struct {
	HeadBranch string `json:"head_branch"`
}

type ghPullRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
}

func main() {
	provisioningSecret = os.Getenv("PROVISIONING_SECRET")
	if provisioningSecret == "" {
		log.Fatal("PROVISIONING_SECRET env var is required")
	}
	githubToken = os.Getenv("GITHUB_TOKEN")

	mux := http.NewServeMux()
	mux.HandleFunc("/provision", handleProvision)
	mux.HandleFunc("/artifacts", handleArtifacts)
	mux.HandleFunc("/artifacts/", handleArtifactsPrefix)

	log.Printf("provisioning service listening on %s", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, mux))
}

func handleProvision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !authenticate(w, r) {
		return
	}
	if !rateCheck(w, r) {
		return
	}
	// Placeholder — existing provisioning logic for Headscale pre-auth keys
	// would go here. Kept as a stub for this PR since it's not part of #886.
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleArtifacts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !authenticate(w, r) {
		return
	}
	if !requireGitHubToken(w) {
		return
	}
	if !rateCheck(w, r) {
		return
	}

	artifacts, err := fetchBranchArtifacts()
	if err != nil {
		log.Printf("error fetching artifacts: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch artifacts"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(artifacts)
}

func handleArtifactsPrefix(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/artifacts/")

	if !strings.HasSuffix(path, "/latest") {
		http.NotFound(w, r)
		return
	}

	branch := strings.TrimSuffix(path, "/latest")
	if branch == "" {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !authenticate(w, r) {
		return
	}
	if !requireGitHubToken(w) {
		return
	}
	if !rateCheck(w, r) {
		return
	}

	handleArtifactDownload(w, r, branch)
}

func handleArtifactDownload(w http.ResponseWriter, r *http.Request, branch string) {
	artifacts, err := fetchBranchArtifacts()
	if err != nil {
		log.Printf("error fetching artifacts: %v", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch artifacts"})
		return
	}

	var target *branchArtifact
	for i := range artifacts {
		if artifacts[i].Branch == branch {
			target = &artifacts[i]
			break
		}
	}
	if target == nil {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": fmt.Sprintf("no artifact found for branch %q", branch)})
		return
	}

	zipData, err := downloadArtifactZip(target.ArtifactID)
	if err != nil {
		log.Printf("error downloading artifact %d: %v", target.ArtifactID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to download artifact"})
		return
	}
	defer os.Remove(zipData)

	binary, err := extractBinaryFromZip(zipData)
	if err != nil {
		log.Printf("error extracting binary from artifact %d: %v", target.ArtifactID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to extract binary"})
		return
	}
	defer binary.Close()

	binaryBytes, err := io.ReadAll(binary)
	if err != nil {
		log.Printf("error reading binary from artifact %d: %v", target.ArtifactID, err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to read binary"})
		return
	}

	hash := sha256.Sum256(binaryBytes)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="autobutler"`)
	w.Header().Set("X-Content-SHA256", hex.EncodeToString(hash[:]))
	w.Write(binaryBytes)
}

func fetchBranchArtifacts() ([]branchArtifact, error) {
	if cached, ok := artifactCache.get(); ok {
		return cached, nil
	}

	url := fmt.Sprintf("%s/repos/%s/actions/artifacts?per_page=100&name=%s", ghAPIBase, repoPath, artifactName)
	var artResp ghArtifactsResponse
	if err := ghGet(url, &artResp); err != nil {
		return nil, fmt.Errorf("fetching artifacts: %w", err)
	}

	branchMap := make(map[string]ghArtifact)
	for _, a := range artResp.Artifacts {
		if a.Expired {
			continue
		}

		var branch string
		if a.WorkflowRun != nil && a.WorkflowRun.HeadBranch != "" {
			branch = a.WorkflowRun.HeadBranch
		} else {
			runURL := fmt.Sprintf("%s/repos/%s/actions/runs/%d", ghAPIBase, repoPath, a.WorkflowRunID)
			var run ghWorkflowRun
			if err := ghGet(runURL, &run); err != nil {
				log.Printf("warning: failed to fetch run %d: %v", a.WorkflowRunID, err)
				continue
			}
			branch = run.HeadBranch
		}

		if branch == "" {
			continue
		}

		if existing, ok := branchMap[branch]; !ok || a.CreatedAt > existing.CreatedAt {
			branchMap[branch] = a
		}
	}

	var results []branchArtifact
	for branch, a := range branchMap {
		ba := branchArtifact{
			Branch:     branch,
			BuiltAt:    a.CreatedAt,
			ArtifactID: a.ID,
		}

		prURL := fmt.Sprintf("%s/repos/%s/pulls?state=open&head=%s:%s", ghAPIBase, repoPath, "autobutler-org", branch)
		var prs []ghPullRequest
		if err := ghGet(prURL, &prs); err == nil && len(prs) > 0 {
			ba.PRNumber = prs[0].Number
			ba.PRTitle = prs[0].Title
		}

		results = append(results, ba)
	}

	artifactCache.set(results)
	return results, nil
}

func downloadArtifactZip(artifactID int64) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/actions/artifacts/%d/zip", ghAPIBase, repoPath, artifactID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+githubToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading zip: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github returned %d: %s", resp.StatusCode, string(body))
	}

	tmp, err := os.CreateTemp("", "artifact-*.zip")
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}
	tmp.Close()

	return tmp.Name(), nil
}

func extractBinaryFromZip(zipPath string) (io.ReadCloser, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("opening zip: %w", err)
	}

	for _, f := range r.File {
		if f.Name == artifactName || (!strings.Contains(f.Name, "/") && f.FileInfo().Mode()&0111 != 0) {
			rc, err := f.Open()
			if err != nil {
				r.Close()
				return nil, fmt.Errorf("opening zip entry %q: %w", f.Name, err)
			}
			return &zipEntryReader{rc: rc, closer: r}, nil
		}
	}

	if len(r.File) > 0 {
		f := r.File[0]
		rc, err := f.Open()
		if err != nil {
			r.Close()
			return nil, fmt.Errorf("opening zip entry %q: %w", f.Name, err)
		}
		return &zipEntryReader{rc: rc, closer: r}, nil
	}

	r.Close()
	return nil, fmt.Errorf("no binary found in zip")
}

type zipEntryReader struct {
	rc     io.ReadCloser
	closer io.Closer
}

func (z *zipEntryReader) Read(p []byte) (int, error) {
	return z.rc.Read(p)
}

func (z *zipEntryReader) Close() error {
	z.rc.Close()
	return z.closer.Close()
}

func ghGet(url string, target interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+githubToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github API %s returned %d: %s", url, resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func authenticate(w http.ResponseWriter, r *http.Request) bool {
	secret := r.Header.Get("X-Provisioning-Secret")
	if secret == "" || subtle.ConstantTimeCompare([]byte(secret), []byte(provisioningSecret)) != 1 {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return false
	}
	return true
}

func requireGitHubToken(w http.ResponseWriter) bool {
	if githubToken == "" {
		jsonResponse(w, http.StatusServiceUnavailable, map[string]string{"error": "github token not configured"})
		return false
	}
	return true
}

func rateCheck(w http.ResponseWriter, r *http.Request) bool {
	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = strings.Split(fwd, ",")[0]
	}
	if !rateLimiter.allow(strings.TrimSpace(ip)) {
		jsonResponse(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
		return false
	}
	return true
}

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
