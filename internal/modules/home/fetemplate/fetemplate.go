// Package fetemplate adalah frontend template switcher (landing) — port PRESISI
// NodeAdmin (katalog opentailwind). Dua sumber template:
//
//   - BUILTIN (default = "agency-consulting-002-creative-agency") → dirender via
//     Go view "home/default" (landing kaya, ter-bundle, jalan offline). Sama dgn
//     NodeAdmin yang merender DEFAULT_FE_TEMPLATE lewat EJS fe/default.
//   - EKSTERNAL (slug pola opentailwind, 640 landing) → HTML diunduh on-demand &
//     di-cache, lalu disajikan sebagai HTML mentah.
//
// File ini bagian MURNI (tanpa jaringan/state): tipe, validasi slug (anti-SSRF),
// derive metadata, builtin, dan katalog kurasi fallback (15, identik NodeAdmin).
package fetemplate

import (
	"regexp"
	"strings"
)

// Template = satu desain landing di katalog.
type Template struct {
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Builtin  bool   `json:"builtin"`
}

// DefaultSlug = template default (ter-bundle, dirender Go view). IDENTIK dgn
// NodeAdmin DEFAULT_FE_TEMPLATE — slug opentailwind "Creative Agency".
const DefaultSlug = "agency-consulting-002-creative-agency"

// slugRe membatasi slug eksternal ke pola opentailwind `{kategori}-{NNN}-{nama}`
// (anti-SSRF: charset & struktur tetap → tak bisa memaksa fetch URL sembarang).
var slugRe = regexp.MustCompile(`^([a-z]+(?:-[a-z]+)*)-(\d{3})-([a-z0-9-]+)$`)

// builtins = SATU template bawaan = default (sama seperti NodeAdmin: hanya
// DEFAULT_FE_TEMPLATE yang di-bundle/EJS, sisanya diunduh on-demand).
var builtins = []Template{
	{Slug: DefaultSlug, Name: "Creative Agency", Category: "Agency", Builtin: true},
}

// builtinViews memetakan slug builtin → nama Go view ("home/<view>"). Default →
// view "default" (landing kaya fe/default). Memungkinkan slug opentailwind
// dirender lewat view Go, sejajar isDefaultEjs NodeAdmin.
var builtinViews = map[string]string{
	DefaultSlug: "default",
}

// BuiltinView mengembalikan nama Go view untuk slug builtin (fallback "default").
func BuiltinView(slug string) string {
	if v, ok := builtinViews[slug]; ok {
		return v
	}
	return "default"
}

// curated = katalog kurasi fallback (15, IDENTIK FE_TEMPLATES NodeAdmin) — dipakai
// saat fetch katalog 640 opentailwind gagal/offline. Default ada di urutan pertama.
var curated = []Template{
	{Slug: "agency-consulting-002-creative-agency", Name: "Creative Agency", Category: "Agency"},
	{Slug: "agency-consulting-001-digital-marketing-agency", Name: "Digital Marketing Agency", Category: "Agency"},
	{Slug: "technology-saas-001-hero-focused-conversion-page", Name: "SaaS — Hero Focused", Category: "Technology"},
	{Slug: "technology-saas-002-feature-rich-multi-section", Name: "SaaS — Feature Rich", Category: "Technology"},
	{Slug: "ecommerce-retail-001-fashion-boutique", Name: "Fashion Boutique", Category: "E-commerce"},
	{Slug: "ecommerce-retail-002-luxury-fashion-brand", Name: "Luxury Fashion", Category: "E-commerce"},
	{Slug: "portfolio-creative-001-creative-portfolio", Name: "Creative Portfolio", Category: "Portfolio"},
	{Slug: "portfolio-creative-002-minimal-portfolio", Name: "Minimal Portfolio", Category: "Portfolio"},
	{Slug: "professional-services-001-law-firm", Name: "Law Firm", Category: "Professional"},
	{Slug: "real-estate-property-001-real-estate-agency", Name: "Real Estate Agency", Category: "Real Estate"},
	{Slug: "food-hospitality-001-fine-dining-restaurant", Name: "Fine Dining", Category: "Food"},
	{Slug: "healthcare-wellness-001-family-doctor-clinic", Name: "Family Clinic", Category: "Healthcare"},
	{Slug: "education-training-001-private-school", Name: "Private School", Category: "Education"},
	{Slug: "fitness-sports-001-fitness-center", Name: "Fitness Center", Category: "Fitness"},
	{Slug: "travel-tourism-001-travel-agency", Name: "Travel Agency", Category: "Travel"},
}

// Builtins mengembalikan salinan template bawaan.
func Builtins() []Template {
	out := make([]Template, len(builtins))
	copy(out, builtins)
	return out
}

// Curated mengembalikan salinan katalog kurasi eksternal (fallback).
func Curated() []Template {
	out := make([]Template, len(curated))
	copy(out, curated)
	return out
}

// IsBuiltin true bila slug = template bawaan GoAdmin.
func IsBuiltin(slug string) bool {
	for _, t := range builtins {
		if t.Slug == slug {
			return true
		}
	}
	return false
}

// IsValidSlug true bila slug = builtin atau cocok pola opentailwind. Inilah
// gerbang ANTI-SSRF: hanya slug valid yang boleh di-fetch/unduh.
func IsValidSlug(slug string) bool {
	return IsBuiltin(slug) || slugRe.MatchString(slug)
}

// ResolveActive mengembalikan slug aktif valid (atau DefaultSlug bila invalid/kosong).
func ResolveActive(slug string) string {
	if IsValidSlug(slug) {
		return slug
	}
	return DefaultSlug
}

// Derive menyusun metadata tampil dari slug opentailwind; bila tak cocok pola,
// pakai slug apa adanya (kategori "Other").
func Derive(slug string) Template {
	m := slugRe.FindStringSubmatch(slug)
	if m == nil {
		return Template{Slug: slug, Name: titleize(slug), Category: "Other"}
	}
	return Template{Slug: slug, Name: titleize(m[3]), Category: titleize(m[1])}
}

// titleize: "digital-marketing" → "Digital Marketing".
func titleize(s string) string {
	parts := strings.Split(s, "-")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		out = append(out, strings.ToUpper(p[:1])+p[1:])
	}
	return strings.Join(out, " ")
}
