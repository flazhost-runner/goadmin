package unit

import (
	"context"
	"testing"

	apperr "goadmin/internal/errors"
	"goadmin/internal/modules/home/fetemplate"
)

// fakeFetcher meniru sumber eksternal tanpa jaringan.
type fakeFetcher struct {
	tree    []fetemplate.Template
	html    string
	treeErr error
	htmlErr error
	htmlHit int
}

func (f *fakeFetcher) Tree(_ context.Context) ([]fetemplate.Template, error) {
	if f.treeErr != nil {
		return nil, f.treeErr
	}
	return f.tree, nil
}
func (f *fakeFetcher) HTML(_ context.Context, _ string) (string, error) {
	f.htmlHit++
	if f.htmlErr != nil {
		return "", f.htmlErr
	}
	return f.html, nil
}

// IsValidSlug = gerbang anti-SSRF: builtin & pola opentailwind lolos; lainnya tidak.
func TestFeTemplate_SlugValidation(t *testing.T) {
	ok := []string{fetemplate.DefaultSlug, "agency-consulting-002-creative-agency", "technology-saas-001-hero", "fitness-sports-001-fitness-center"}
	bad := []string{"../etc/passwd", "http://evil", "Foo", "bar", "no-number-here", "a/b"}
	for _, s := range ok {
		if !fetemplate.IsValidSlug(s) {
			t.Fatalf("%q seharusnya valid", s)
		}
	}
	for _, s := range bad {
		if fetemplate.IsValidSlug(s) {
			t.Fatalf("%q seharusnya DITOLAK (anti-SSRF)", s)
		}
	}
}

// Tanpa fetcher → katalog = builtin + kurasi (tanpa jaringan).
func TestFeTemplate_CatalogFallback(t *testing.T) {
	svc := fetemplate.New(nil, t.TempDir())
	cat := svc.Catalog(context.Background())
	if len(cat) < 3 {
		t.Fatalf("katalog terlalu sedikit: %d", len(cat))
	}
	// Builtin (default = Creative Agency) di depan.
	if !cat[0].Builtin || cat[0].Slug != fetemplate.DefaultSlug {
		t.Fatalf("builtin harus di depan, dapat %+v", cat[0])
	}
}

// Paginate menyaring + menyematkan slug aktif ke halaman pertama.
func TestFeTemplate_PaginatePinAndSearch(t *testing.T) {
	svc := fetemplate.New(nil, t.TempDir())
	ctx := context.Background()

	// Pin slug kurasi → item pertama.
	pin := "travel-tourism-001-travel-agency"
	items, total := svc.Paginate(ctx, "", "", 1, 3, pin)
	if total < 3 || items[0].Slug != pin {
		t.Fatalf("pin gagal: first=%s total=%d", items[0].Slug, total)
	}

	// Search 'agency' → semua hasil mengandung 'agency'.
	hit, n := svc.Paginate(ctx, "agency", "", 1, 50, "")
	if n == 0 {
		t.Fatal("search 'agency' kosong")
	}
	for _, it := range hit {
		if !contains(it.Name+" "+it.Slug, "agency") && !contains(it.Name+" "+it.Slug, "Agency") {
			t.Fatalf("hasil tak relevan: %s", it.Slug)
		}
	}
}

// Ensure: builtin no-op; slug invalid ditolak; tanpa fetcher eksternal → 502.
func TestFeTemplate_EnsureGate(t *testing.T) {
	svc := fetemplate.New(nil, t.TempDir())
	ctx := context.Background()

	if err := svc.Ensure(ctx, fetemplate.DefaultSlug); err != nil {
		t.Fatalf("builtin harus no-op: %v", err)
	}
	if err := svc.Ensure(ctx, "../evil"); err == nil {
		t.Fatal("slug invalid harus ditolak")
	}
	// Slug eksternal (bukan builtin) tanpa fetcher → 502.
	err := svc.Ensure(ctx, "technology-saas-002-feature-rich-multi-section")
	if ae, ok := apperr.As(err); !ok || ae.Status != 502 {
		t.Fatalf("tanpa remote harus 502, dapat: %v", err)
	}
}

// Dengan fetcher: Ensure mengunduh + cache; pemanggilan kedua TAK fetch lagi.
func TestFeTemplate_EnsureDownloadsAndCaches(t *testing.T) {
	f := &fakeFetcher{html: "<html><body>Landing X</body></html>"}
	svc := fetemplate.New(f, t.TempDir())
	ctx := context.Background()
	slug := "technology-saas-001-hero-focused-conversion-page"

	if err := svc.Ensure(ctx, slug); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if f.htmlHit != 1 {
		t.Fatalf("harus fetch sekali, dapat %d", f.htmlHit)
	}
	// Sudah ter-cache → ensure kedua tak fetch.
	if err := svc.Ensure(ctx, slug); err != nil {
		t.Fatalf("ensure2: %v", err)
	}
	if f.htmlHit != 1 {
		t.Fatalf("ensure kedua tak boleh fetch lagi, hit=%d", f.htmlHit)
	}
	// ActiveHTML mengembalikan HTML ter-cache.
	html, err := svc.ActiveHTML(ctx, slug)
	if err != nil || !contains(html, "Landing X") {
		t.Fatalf("active html salah: %v / %q", err, html)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (indexOf(s, sub) >= 0)
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
