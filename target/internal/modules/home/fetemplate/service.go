package fetemplate

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	apperr "goadmin/internal/errors"
)

// catalogTTL = masa berlaku cache memori katalog eksternal.
const catalogTTL = 6 * time.Hour

// Service mengelola katalog (builtin + eksternal) + unduh/cache HTML.
// fetcher nil → mode non-remote (hanya katalog kurasi, tanpa jaringan).
type Service struct {
	fetcher  Fetcher
	cacheDir string

	mu     sync.Mutex
	memo   []Template
	memoAt time.Time
}

// New merakit service. fetcher boleh nil (remote nonaktif).
func New(fetcher Fetcher, cacheDir string) *Service {
	return &Service{fetcher: fetcher, cacheDir: cacheDir}
}

// Catalog = builtin + eksternal (dedup; builtin di depan).
func (s *Service) Catalog(ctx context.Context) []Template {
	out := Builtins()
	seen := make(map[string]bool, len(out))
	for _, t := range out {
		seen[t.Slug] = true
	}
	for _, t := range s.external(ctx) {
		if !seen[t.Slug] {
			out = append(out, t)
			seen[t.Slug] = true
		}
	}
	return out
}

// external mengembalikan daftar eksternal: cache memori (TTL) → fetch → disk →
// kurasi. Sekali fetch lalu di-memo agar tak membebani sumber.
func (s *Service) external(ctx context.Context) []Template {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.memo != nil && time.Since(s.memoAt) < catalogTTL {
		return s.memo
	}
	if s.fetcher != nil {
		if data, err := s.fetcher.Tree(ctx); err == nil && len(data) > 0 {
			s.memo, s.memoAt = data, time.Now()
			s.writeDiskCatalog(data)
			return data
		}
		if disk := s.readDiskCatalog(); disk != nil {
			s.memo, s.memoAt = disk, time.Now()
			return disk
		}
	}
	s.memo, s.memoAt = Curated(), time.Now()
	return s.memo
}

// Paginate menyaring (nama/slug + kategori), menyematkan slug aktif ke depan,
// lalu memotong per halaman. Mengembalikan item + total hasil filter.
func (s *Service) Paginate(ctx context.Context, qName, qCategory string, page, pageSize int, pin string) (items []Template, total int) {
	all := s.Catalog(ctx)
	name := strings.ToLower(strings.TrimSpace(qName))
	cat := strings.TrimSpace(qCategory)

	filtered := make([]Template, 0, len(all))
	for _, t := range all {
		okName := name == "" || strings.Contains(strings.ToLower(t.Name), name) || strings.Contains(strings.ToLower(t.Slug), name)
		okCat := cat == "" || t.Category == cat
		if okName && okCat {
			filtered = append(filtered, t)
		}
	}

	// Sematkan template aktif ke paling depan (tampil di halaman pertama).
	if pin != "" {
		for i, t := range filtered {
			if t.Slug == pin && i > 0 {
				p := filtered[i]
				copy(filtered[1:i+1], filtered[0:i])
				filtered[0] = p
				break
			}
		}
	}

	total = len(filtered)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 12
	}
	start := (page - 1) * pageSize
	if start >= total {
		return []Template{}, total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return filtered[start:end], total
}

// Categories mengembalikan kategori unik terurut.
func (s *Service) Categories(ctx context.Context) []string {
	set := map[string]bool{}
	for _, t := range s.Catalog(ctx) {
		set[t.Category] = true
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Ensure memastikan HTML template tersedia lokal (unduh bila perlu). Builtin →
// no-op (dirender Go view). Slug invalid ditolak (anti-SSRF).
func (s *Service) Ensure(ctx context.Context, slug string) error {
	if !IsValidSlug(slug) {
		return apperr.BadRequest("Template tidak dikenali")
	}
	if IsBuiltin(slug) {
		return nil
	}
	if _, ok := s.CachedHTML(slug); ok {
		return nil
	}
	if s.fetcher == nil {
		return apperr.New(http.StatusBadGateway, "Unduhan butuh mode remote (FE_TEMPLATE_REMOTE=true)")
	}
	html, err := s.fetcher.HTML(ctx, slug)
	if err != nil {
		return apperr.New(http.StatusBadGateway, "Gagal mengunduh template: "+err.Error())
	}
	if err := s.writeHTML(slug, html); err != nil {
		return apperr.Internal(err.Error())
	}
	return nil
}

// ActiveHTML mengembalikan HTML landing eksternal aktif (cache lokal → unduh →
// error). Builtin TIDAK lewat sini (controller render Go view).
func (s *Service) ActiveHTML(ctx context.Context, slug string) (string, error) {
	if html, ok := s.CachedHTML(slug); ok {
		return html, nil
	}
	if s.fetcher != nil {
		if err := s.Ensure(ctx, slug); err == nil {
			if html, ok := s.CachedHTML(slug); ok {
				return html, nil
			}
		}
	}
	return "", apperr.New(http.StatusBadGateway, "Template belum tersedia")
}

// PreviewHTML menyajikan HTML untuk pratinjau: cache lokal lebih dulu (instan),
// lalu fetch upstream (timeout), lalu fallback lokal. Anti-SSRF: slug divalidasi.
func (s *Service) PreviewHTML(ctx context.Context, slug string) (string, error) {
	if !IsValidSlug(slug) {
		return "", apperr.BadRequest("Template tidak dikenali")
	}
	if html, ok := s.CachedHTML(slug); ok {
		return html, nil
	}
	if s.fetcher != nil {
		if html, err := s.fetcher.HTML(ctx, slug); err == nil {
			return html, nil
		}
	}
	return "", apperr.New(http.StatusBadGateway, "Gagal mengambil preview")
}

// --- cache lokal (disk) ---

func (s *Service) htmlFile(slug string) string { return filepath.Join(s.cacheDir, slug+".html") }
func (s *Service) catalogFile() string         { return filepath.Join(s.cacheDir, "_catalog.json") }

// CachedHTML membaca HTML template dari cache lokal (valid bila memuat </html>).
func (s *Service) CachedHTML(slug string) (string, bool) {
	b, err := os.ReadFile(s.htmlFile(slug))
	if err != nil {
		return "", false
	}
	html := string(b)
	if !strings.Contains(strings.ToLower(html), "</html>") {
		return "", false
	}
	return html, true
}

func (s *Service) writeHTML(slug, html string) error {
	if err := os.MkdirAll(s.cacheDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.htmlFile(slug), []byte(html), 0o644)
}

func (s *Service) readDiskCatalog() []Template {
	b, err := os.ReadFile(s.catalogFile())
	if err != nil {
		return nil
	}
	var data []Template
	if json.Unmarshal(b, &data) != nil || len(data) == 0 {
		return nil
	}
	return data
}

func (s *Service) writeDiskCatalog(data []Template) {
	if err := os.MkdirAll(s.cacheDir, 0o755); err != nil {
		return // best-effort
	}
	if b, err := json.Marshal(data); err == nil {
		_ = os.WriteFile(s.catalogFile(), b, 0o644)
	}
}
