// Command make-module men-scaffold satu modul CRUD lengkap yang OTOMATIS
// mengikuti pola modul referensi `access` & lolos convention checker —
// padanan skill `/make-module` di NodeAdmin.
//
// Menghasilkan: model + migration + dto + service(+interface) + controller
// (api[+web]) + route(api[+web]) + module.go + integration test, lalu
// mendaftarkan blank-import modul ke internal/modules/modules.go.
//
// Pakai:
//
//	go run ./cmd/make-module --name product            # full (api + web)
//	go run ./cmd/make-module --name token --web=false  # api-only
//	go run ./cmd/make-module --name category --plural categories
//
// Setelah generate, verifikasi: `make verify`.
package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// data adalah konteks yang diumpankan ke tiap template.
type data struct {
	Module     string // nama paket modul (lowercase singular), mis. "product"
	Type       string // nama tipe PascalCase, mis. "Product"
	Plural     string // bentuk jamak untuk path/tabel, mis. "products"
	Table      string // nama tabel DB (= Plural)
	CodePrefix string // prefix kode entity, mis. "P"
	WantWeb    bool   // ikut hasilkan lapisan web (UI admin)
}

// fileSpec memetakan satu template → path output (relatif root modul).
type fileSpec struct {
	tmpl    string // nama file template di templates/
	out     string // path output relatif root project
	webOnly bool   // hanya dibuat bila WantWeb
}

var nameRe = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

func main() {
	name := flag.String("name", "", "nama modul (lowercase, singular, satu kata) — wajib")
	plural := flag.String("plural", "", "bentuk jamak untuk path/tabel (default: name+heuristik)")
	web := flag.Bool("web", true, "ikut hasilkan lapisan web/UI admin (set false untuk API-only)")
	force := flag.Bool("force", false, "timpa bila folder modul sudah ada")
	flag.Parse()

	if err := run(*name, *plural, *web, *force); err != nil {
		fmt.Fprintln(os.Stderr, "make-module: "+err.Error())
		os.Exit(1)
	}
}

func run(name, plural string, web, force bool) error {
	root, err := projectRoot()
	if err != nil {
		return err
	}

	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return fmt.Errorf("flag --name wajib (mis. --name product)")
	}
	if !nameRe.MatchString(name) {
		return fmt.Errorf("nama '%s' tak valid — harus lowercase, mulai huruf, tanpa spasi/strip/underscore (nama paket Go)", name)
	}
	if name == "access" || name == "model" || name == "service" || name == "dto" {
		return fmt.Errorf("nama '%s' bentrok dengan modul/paket inti — pilih nama lain", name)
	}

	d := data{
		Module:     name,
		Type:       pascal(name),
		Plural:     pickPlural(name, plural),
		CodePrefix: strings.ToUpper(name[:1]),
		WantWeb:    web,
	}
	d.Table = d.Plural

	moduleDir := filepath.Join(root, "internal", "modules", name)
	if _, err := os.Stat(moduleDir); err == nil && !force {
		return fmt.Errorf("folder modul sudah ada: %s (pakai --force untuk menimpa)", rel(root, moduleDir))
	}

	specs := []fileSpec{
		{"model.tmpl", filepath.Join("internal/modules", name, "model", name+".go"), false},
		{"migration.tmpl", filepath.Join("internal/modules", name, "migration", "automigrate.go"), false},
		{"dto.tmpl", filepath.Join("internal/modules", name, "dto", name+"_dto.go"), false},
		{"interfaces.tmpl", filepath.Join("internal/modules", name, "service", "interfaces.go"), false},
		{"service.tmpl", filepath.Join("internal/modules", name, "service", name+"_service.go"), false},
		{"controller_api.tmpl", filepath.Join("internal/modules", name, "controller", "api", name+"_controller.go"), false},
		{"controller_web.tmpl", filepath.Join("internal/modules", name, "controller", "web", name+"_controller.go"), true},
		{"route_api.tmpl", filepath.Join("internal/modules", name, "route_api.go"), false},
		{"route_web.tmpl", filepath.Join("internal/modules", name, "route_web.go"), true},
		{"module.tmpl", filepath.Join("internal/modules", name, "module.go"), false},
		{"test.tmpl", filepath.Join("tests/integration", name+"_service_test.go"), false},
	}

	var written []string
	for _, s := range specs {
		if s.webOnly && !d.WantWeb {
			continue
		}
		out := filepath.Join(root, s.out)
		if err := renderToFile(s.tmpl, out, d); err != nil {
			return fmt.Errorf("render %s: %w", s.tmpl, err)
		}
		written = append(written, s.out)
	}

	if err := registerImport(root, name); err != nil {
		return fmt.Errorf("daftarkan import ke modules.go: %w", err)
	}

	report(d, written)
	return nil
}

// renderToFile mengeksekusi template, mem-format (gofmt), lalu menulis file.
func renderToFile(tmplName, outPath string, d data) error {
	raw, err := templatesFS.ReadFile("templates/" + tmplName)
	if err != nil {
		return err
	}
	tmpl, err := template.New(tmplName).Parse(string(raw))
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, d); err != nil {
		return err
	}

	out := buf.Bytes()
	if strings.HasSuffix(outPath, ".go") {
		formatted, ferr := format.Source(out)
		if ferr != nil {
			return fmt.Errorf("gofmt gagal (template rusak?): %w", ferr)
		}
		out = formatted
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outPath, out, 0o644)
}

// registerImport menambahkan blank-import modul ke internal/modules/modules.go
// (idempotent — dilewati bila sudah ada).
func registerImport(root, name string) error {
	path := filepath.Join(root, "internal", "modules", "modules.go")
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	importPath := "goadmin/internal/modules/" + name
	if strings.Contains(string(src), `"`+importPath+`"`) {
		return nil // sudah terdaftar
	}

	lines := strings.Split(string(src), "\n")
	out := make([]string, 0, len(lines)+1)
	inserted := false
	inImport := false
	for _, ln := range lines {
		if strings.HasPrefix(strings.TrimSpace(ln), "import (") {
			inImport = true
		}
		if inImport && !inserted && strings.TrimSpace(ln) == ")" {
			out = append(out, "\t_ \""+importPath+"\"")
			inserted = true
			inImport = false
		}
		out = append(out, ln)
	}
	if !inserted {
		return fmt.Errorf("blok import tak ditemukan di modules.go")
	}

	formatted, err := format.Source([]byte(strings.Join(out, "\n")))
	if err != nil {
		return fmt.Errorf("gofmt modules.go: %w", err)
	}
	return os.WriteFile(path, formatted, 0o644)
}

// projectRoot mengembalikan cwd, memastikan ini root modul goadmin.
func projectRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	gomod, err := os.ReadFile(filepath.Join(wd, "go.mod"))
	if err != nil || !strings.Contains(string(gomod), "module goadmin") {
		return "", fmt.Errorf("jalankan dari root project GoAdmin (folder berisi go.mod 'module goadmin')")
	}
	return wd, nil
}

func report(d data, written []string) {
	fmt.Printf("\n✅ Modul '%s' (tipe %s) dibuat:\n\n", d.Module, d.Type)
	for _, f := range written {
		fmt.Println("  + " + f)
	}
	fmt.Println("  ~ internal/modules/modules.go (blank-import didaftarkan)")

	fmt.Println("\nLangkah lanjut:")
	fmt.Println("  1. make verify        # checker + vet + build + test (harus hijau)")
	if d.WantWeb {
		fmt.Printf("  2. Buat view HTML di internal/modules/%s/view/index.html (render \"%s/index\").\n", d.Module, d.Module)
		fmt.Printf("  3. Seed permission '%s.view/.create/.update/.delete' agar RBAC mengizinkan akses.\n", d.Module)
	} else {
		fmt.Printf("  2. Seed permission '%s.view/.create/.update/.delete' agar RBAC mengizinkan akses.\n", d.Module)
	}
	fmt.Printf("  -  Endpoint API: /api/v1/%s (index/show/store/update/destroy).\n", d.Plural)
	fmt.Println()
}

// --- util nama ---

func pascal(s string) string {
	return strings.ToUpper(s[:1]) + s[1:]
}

// pickPlural mengembalikan plural eksplisit bila diberi, atau heuristik sederhana.
func pickPlural(name, explicit string) string {
	if e := strings.ToLower(strings.TrimSpace(explicit)); e != "" {
		return e
	}
	return pluralize(name)
}

// pluralize: heuristik EN sederhana (cukup untuk nama path/tabel; override via --plural).
func pluralize(s string) string {
	switch {
	case strings.HasSuffix(s, "y") && len(s) > 1 && !isVowel(s[len(s)-2]):
		return s[:len(s)-1] + "ies"
	case strings.HasSuffix(s, "s"), strings.HasSuffix(s, "x"), strings.HasSuffix(s, "z"),
		strings.HasSuffix(s, "ch"), strings.HasSuffix(s, "sh"):
		return s + "es"
	default:
		return s + "s"
	}
}

func isVowel(b byte) bool { return strings.IndexByte("aeiou", b) >= 0 }

func rel(root, p string) string {
	if r, err := filepath.Rel(root, p); err == nil {
		return r
	}
	return p
}
