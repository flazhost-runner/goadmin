// Command checkconventions adalah convention checker GoAdmin — padanan
// `nodeadmin check` (packages/cli/lib/checkConventions.js) di NodeAdmin.
//
// Tujuannya menjaga modul baru sejalan dengan pola & prinsip yang sudah
// ditetapkan (SOLID/DI, error handling terpusat, portabilitas DB, security).
// Berbeda dari versi NodeAdmin yang berbasis regex, versi ini memakai go/ast
// agar akurat: hanya menilai kode nyata (call/struct-tag/deklarasi), bukan
// komentar atau string yang kebetulan mirip.
//
// Jalankan: `go run ./cmd/checkconventions` (lokal) & di CI.
// Exit 1 bila ada pelanggaran (gate).
//
// Pemetaan aturan NodeAdmin → GoAdmin:
//
//	instanceof Error / return error  → service dilarang bikin error telanjang
//	                                   (errors.New/fmt.Errorf) → pakai apperr.*
//	route new XService()            → controller dilarang `service.New*`/literal
//	                                   service konkret (DI lewat container)
//	service @injectable+implements  → tiap *_service.go wajib assertion compile
//	                                   `var _ IXxx = (*Xxx)(nil)`
//	web res.render(path)            → web controller dilarang `c.HTML(` →
//	                                   view.RenderView
//	entity tipe portabel            → gorm tag model larang longtext/datetime/
//	                                   enum/collation
//	modules raw .query()/LIKE :     → modules larang db.Raw/Exec + literal LIKE
//	                                   → helpers.CiLike
//	modules process.env             → modules larang os.Getenv → config
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// violation adalah satu pelanggaran prinsip/pola (memblok, exit 1).
type violation struct {
	file string
	line int
	msg  string
}

// warning bersifat informatif (tidak memblok).
type warning struct {
	file string
	msg  string
}

var (
	violations []violation
	warnings   []warning
)

func violate(file string, line int, msg string) {
	violations = append(violations, violation{file: file, line: line, msg: msg})
}
func warn(file, msg string) { warnings = append(warnings, warning{file: file, msg: msg}) }

// serviceFileRe mencocokkan file service konkret (mis. user_service.go),
// mengecualikan interfaces.go.
var serviceFileRe = regexp.MustCompile(`_service\.go$`)

// nonPortableType = tipe kolom/atribut gorm yang vendor-spesifik & tak portabel
// lintas dialek (MySQL/PG/SQLite).
var nonPortableType = []string{"longtext", "mediumtext", "tinytext", "datetime", "enum", "collation", "collate"}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "checkconventions: gagal baca cwd:", err)
		os.Exit(2)
	}

	scanDir := filepath.Join(root, "internal")
	if _, err := os.Stat(scanDir); err != nil {
		fmt.Fprintln(os.Stderr, "checkconventions: folder internal/ tak ditemukan — jalankan dari root modul.")
		os.Exit(2)
	}

	fset := token.NewFileSet()
	walkGoFiles(scanDir, func(path string) {
		checkFile(fset, root, path)
	})

	checkModuleTests(root)
	report()
}

// walkGoFiles memanggil cb untuk tiap *.go non-test, melewati vendor, folder
// migration (boleh raw SQL), dan file generated.
func walkGoFiles(dir string, cb func(path string)) {
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			// migration/migrations dikecualikan: DDL/seed kadang butuh raw SQL.
			if name == "vendor" || name == ".git" || name == "migration" || name == "migrations" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		cb(path)
		return nil
	})
}

// checkFile mem-parse satu file & menerapkan aturan sesuai perannya.
func checkFile(fset *token.FileSet, root, path string) {
	rel, _ := filepath.Rel(root, path)
	rel = filepath.ToSlash(rel)

	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		// File tak bisa di-parse → biarkan compiler/tsc yang melapor; jangan dobel.
		return
	}

	inModules := strings.Contains(rel, "internal/modules/")
	isService := inModules && strings.Contains(rel, "/service/") && serviceFileRe.MatchString(rel)
	isController := inModules && strings.Contains(rel, "/controller/")
	isWebController := isController && strings.Contains(rel, "/controller/web/")
	isModel := inModules && strings.Contains(rel, "/model/")

	lineOf := func(p token.Pos) int { return fset.Position(p).Line }

	// --- Aturan 3: service wajib assertion compile `var _ IXxx = (*Xxx)(nil)` ---
	if isService {
		checkServiceAssertion(rel, path, file, lineOf)
	}

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {

		case *ast.CallExpr:
			sel, ok := node.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg := identName(sel.X)
			fn := sel.Sel.Name

			// --- Aturan 1: service dilarang bikin error telanjang ---
			if isService {
				if (pkg == "errors" && fn == "New") || (pkg == "fmt" && fn == "Errorf") {
					violate(rel, lineOf(node.Pos()),
						"Service dilarang membuat error telanjang ("+pkg+"."+fn+") — pakai konstruktor apperr.* (NotFound/Conflict/Validation/Internal/…). Idiom Go `return err` atas error yang sudah ada tetap boleh.")
				}
			}

			// --- Aturan 2: controller dilarang meng-instansiasi service konkret ---
			if isController && pkg == "service" && strings.HasPrefix(fn, "New") {
				violate(rel, lineOf(node.Pos()),
					"Controller dilarang `service."+fn+"(` — service di-inject lewat konstruktor (DI dari container/module), bukan dirakit di controller.")
			}

			// --- Aturan 4: web controller dilarang c.HTML( langsung ---
			if isWebController && fn == "HTML" {
				violate(rel, lineOf(node.Pos()),
					"Gunakan `view.RenderView(c, \"modul/view\", gin.H{...})` — bukan `c.HTML(` path mentah (konsistensi layout + enrich currentUser).")
			}

			// --- Aturan 6: modules larang raw SQL db.Raw/Exec ---
			if inModules && (fn == "Raw" || fn == "Exec") {
				violate(rel, lineOf(node.Pos()),
					"Hindari `."+fn+"(` (raw SQL) di modul — rawan sintaks spesifik-DB. Pakai query builder GORM / repository.")
			}

			// --- Aturan 7: modules larang os.Getenv ---
			if inModules && pkg == "os" && fn == "Getenv" {
				violate(rel, lineOf(node.Pos()),
					"Jangan `os.Getenv` di modul — akses konfigurasi lewat config yang di-inject (config.Config).")
			}

		case *ast.CompositeLit:
			// --- Aturan 2 (varian): controller dilarang `service.XxxService{...}` ---
			if isController {
				if sel, ok := node.Type.(*ast.SelectorExpr); ok && identName(sel.X) == "service" {
					violate(rel, lineOf(node.Pos()),
						"Controller dilarang merakit `service."+sel.Sel.Name+"{}` langsung — terima `service.I*Service` lewat konstruktor (DI).")
				}
			}

		case *ast.BasicLit:
			// --- Aturan 6: modules larang literal LIKE manual ---
			if inModules && node.Kind == token.STRING {
				if s, err := strconv.Unquote(node.Value); err == nil && strings.Contains(strings.ToUpper(s), "LIKE") {
					violate(rel, lineOf(node.Pos()),
						"Jangan tulis `LIKE` manual di modul — case-sensitivity berbeda antar-dialek. Pakai helper `helpers.CiLike/CiLikeAny` (LOWER(..) LIKE LOWER(..)).")
				}
			}

		case *ast.StructType:
			// --- Aturan 5: model wajib tipe gorm portabel ---
			if isModel {
				checkModelTags(rel, node, lineOf)
			}
		}
		return true
	})
}

// checkServiceAssertion memastikan ada `var _ IXxxService = (*XxxService)(nil)`
// yang menjamin (saat compile) service mengimplementasi interfacenya.
func checkServiceAssertion(rel, path string, file *ast.File, lineOf func(token.Pos) int) {
	// Tipe service diturunkan dari nama file: user_service.go → UserService.
	base := strings.TrimSuffix(filepath.Base(path), "_service.go")
	typeName := snakeToPascal(base) + "Service"
	ifaceName := "I" + typeName

	found := false
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) != 1 || vs.Names[0].Name != "_" || vs.Type == nil {
				continue
			}
			// Tipe deklarasi harus interfacenya: `var _ IXxxService = ...`.
			if identName(vs.Type) == ifaceName {
				found = true
			}
		}
	}
	if !found {
		violate(rel, 1, fmt.Sprintf(
			"Service wajib assertion compile `var _ %s = (*%s)(nil)` (Dependency Inversion — menjamin implementasi kontrak saat compile).",
			ifaceName, typeName))
	}
}

// checkModelTags menolak tipe kolom gorm yang tak portabel lintas dialek.
func checkModelTags(rel string, st *ast.StructType, lineOf func(token.Pos) int) {
	for _, field := range st.Fields.List {
		if field.Tag == nil {
			continue
		}
		raw, err := strconv.Unquote(field.Tag.Value)
		if err != nil {
			continue
		}
		gormTag := reflect.StructTag(raw).Get("gorm")
		if gormTag == "" {
			continue
		}
		lower := strings.ToLower(gormTag)
		for _, bad := range nonPortableType {
			if strings.Contains(lower, bad) {
				violate(rel, lineOf(field.Pos()), fmt.Sprintf(
					"Tag gorm mengandung `%s` yang tak portabel lintas dialek — pakai tipe abstrak (varchar/text/bigint, time.Time untuk waktu) tanpa collation vendor.",
					bad))
				break
			}
		}
	}
}

// checkModuleTests memperingatkan modul ber-service yang belum punya test
// yang menyebut namanya (heuristik, sejajar warning NodeAdmin).
func checkModuleTests(root string) {
	modulesDir := filepath.Join(root, "internal", "modules")
	testsDir := filepath.Join(root, "tests")
	mods, err := os.ReadDir(modulesDir)
	if err != nil {
		return
	}

	var testNames []string
	_ = filepath.WalkDir(testsDir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".go") {
			testNames = append(testNames, strings.ToLower(filepath.Base(path)))
		}
		return nil
	})

	for _, m := range mods {
		if !m.IsDir() {
			continue
		}
		serviceDir := filepath.Join(modulesDir, m.Name(), "service")
		// Modul tanpa folder service (mis. dashboard read-only) dikecualikan.
		if _, err := os.Stat(serviceDir); err != nil {
			continue
		}

		// Subjek pencocokan test = nama modul + nama tiap service (tanpa _service)
		// + nama tiap model. Test biasanya dinamai per-subjek (user/role/auth),
		// bukan nama modul — selaras heuristik NodeAdmin.
		subjects := map[string]struct{}{strings.ToLower(m.Name()): {}}
		for _, base := range goFileBases(serviceDir, "_service.go") {
			subjects[strings.ToLower(base)] = struct{}{}
		}
		for _, base := range goFileBases(filepath.Join(modulesDir, m.Name(), "model"), ".go") {
			subjects[strings.ToLower(base)] = struct{}{}
		}

		covered := false
		for _, t := range testNames {
			for s := range subjects {
				if strings.Contains(t, s) {
					covered = true
					break
				}
			}
			if covered {
				break
			}
		}
		if !covered {
			warn(filepath.Join("internal", "modules", m.Name()),
				"modul '"+m.Name()+"' belum punya file test yang menyebut modul/service/model-nya di tests/.")
		}
	}
}

// goFileBases mengembalikan nama file (tanpa suffix) di dir yang berakhiran
// suffix, mengecualikan file interface & test. Mis. user_service.go → "user".
func goFileBases(dir, suffix string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, suffix) || strings.HasSuffix(n, "_test.go") || n == "interfaces.go" {
			continue
		}
		out = append(out, strings.TrimSuffix(n, suffix))
	}
	return out
}

// identName mengembalikan nama identifier (atau "" bila bukan ident sederhana).
func identName(e ast.Expr) string {
	if id, ok := e.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

// snakeToPascal: "user" → "User", "password_reset" → "PasswordReset".
func snakeToPascal(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

// report mencetak semua temuan sekaligus lalu exit (1 bila ada pelanggaran).
func report() {
	if len(warnings) > 0 {
		fmt.Println("\n⚠️  Warnings:")
		for _, w := range warnings {
			fmt.Printf("  - %s: %s\n", w.file, w.msg)
		}
	}

	if len(violations) > 0 {
		fmt.Fprintf(os.Stderr, "\n❌ Pelanggaran PRINSIP/POLA (%d):\n\n", len(violations))
		for _, v := range violations {
			fmt.Fprintf(os.Stderr, "  %s:%d\n     → %s\n", v.file, v.line, v.msg)
		}
		fmt.Fprintln(os.Stderr, "\nPerbaiki sesuai AGENTS.md / docs/MODULE_GUIDE.md sebelum melanjutkan.")
		os.Exit(1)
	}

	fmt.Println("\n✅ Konvensi terpenuhi — modul sejalan dengan pola yang ditetapkan.")
}
