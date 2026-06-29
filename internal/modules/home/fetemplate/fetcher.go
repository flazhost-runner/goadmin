package fetemplate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Fetcher menyuplai data eksternal (di-abstraksi agar bisa di-fake di test —
// tak ada jaringan saat `go test`).
type Fetcher interface {
	// Tree mengembalikan daftar template eksternal dari sumber.
	Tree(ctx context.Context) ([]Template, error)
	// HTML mengembalikan HTML mentah satu template (untuk unduh/preview).
	HTML(ctx context.Context, slug string) (string, error)
}

const (
	treeTimeout = 20 * time.Second // tree besar (640) → longgar; hanya sekali lalu di-cache
	htmlTimeout = 8 * time.Second  // 1 file ringan → ketat
	htmlMaxSize = 2 << 20          // 2 MB cap per template
)

// HTTPFetcher mengambil katalog + HTML dari opentailwind (GitHub).
type HTTPFetcher struct {
	client  *http.Client
	treeURL string
	rawBase string
}

// NewHTTPFetcher merakit fetcher HTTP.
func NewHTTPFetcher(treeURL, rawBase string) *HTTPFetcher {
	return &HTTPFetcher{client: &http.Client{}, treeURL: treeURL, rawBase: rawBase}
}

// gitTree = subset respons GitHub git-trees (recursive).
type gitTree struct {
	Tree []struct {
		Path string `json:"path"`
		Type string `json:"type"`
	} `json:"tree"`
}

// Tree mem-fetch daftar landing (`landings/*.html`) dari GitHub tree API.
func (f *HTTPFetcher) Tree(ctx context.Context) ([]Template, error) {
	ctx, cancel := context.WithTimeout(ctx, treeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.treeURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	res, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tree HTTP %d", res.StatusCode)
	}

	var gt gitTree
	if err := json.NewDecoder(res.Body).Decode(&gt); err != nil {
		return nil, err
	}
	out := make([]Template, 0, len(gt.Tree))
	for _, n := range gt.Tree {
		if n.Type == "blob" && strings.HasPrefix(n.Path, "landings/") && strings.HasSuffix(n.Path, ".html") {
			slug := strings.TrimSuffix(strings.TrimPrefix(n.Path, "landings/"), ".html")
			out = append(out, Derive(slug))
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("katalog kosong")
	}
	return out, nil
}

// HTML mem-fetch HTML mentah satu template (timeout ketat + cap ukuran).
func (f *HTTPFetcher) HTML(ctx context.Context, slug string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, htmlTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.rawBase+"/"+slug+".html", nil)
	if err != nil {
		return "", err
	}
	res, err := f.client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("html HTTP %d", res.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, htmlMaxSize))
	if err != nil {
		return "", err
	}
	html := string(b)
	if !strings.Contains(strings.ToLower(html), "</html>") {
		return "", fmt.Errorf("HTML tidak valid")
	}
	return html, nil
}
