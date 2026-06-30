package router

import (
	"sort"
	"sync"

	"github.com/gin-gonic/gin"

	"goadmin/internal/config"
	"goadmin/internal/container"
)

// RegistrationContext diberikan ke tiap modul saat registrasi.
type RegistrationContext struct {
	Mode      config.AppMode
	Container *container.Container
	// Web adalah grup route web (HTML, sesi, CSRF). NIL pada mode api —
	// modul UI HARUS guard: bila ctx.Web == nil, lewati registrasi web diam-diam.
	// Inilah mekanisme "diff purely-additive": modul UI absen di mode api.
	Web *gin.RouterGroup
	// API adalah grup route REST (JSON, JWT). Selalu ada di kedua mode.
	API *gin.RouterGroup
}

// Module adalah kontrak satu fitur (padanan Module.ts + routes/{web,api}.ts).
type Module interface {
	// Name mengembalikan nama modul (untuk log/registry).
	Name() string
	// Register memasang service ke container + route ke grup yang relevan.
	Register(ctx *RegistrationContext)
}

type moduleRegistry struct {
	mu      sync.Mutex
	modules []Module
}

var modReg = &moduleRegistry{}

// Add mendaftarkan modul ke registry global (dipanggil dari init modul / main).
func Add(m Module) {
	modReg.mu.Lock()
	defer modReg.mu.Unlock()
	modReg.modules = append(modReg.modules, m)
}

// RegisterAll menjalankan Register tiap modul terhadap ctx (urut nama agar deterministik).
func RegisterAll(ctx *RegistrationContext) {
	modReg.mu.Lock()
	mods := make([]Module, len(modReg.modules))
	copy(mods, modReg.modules)
	modReg.mu.Unlock()

	sort.Slice(mods, func(i, j int) bool { return mods[i].Name() < mods[j].Name() })
	for _, m := range mods {
		m.Register(ctx)
	}
}

// Names mengembalikan daftar nama modul terdaftar (untuk diagnostik).
func Names() []string {
	modReg.mu.Lock()
	defer modReg.mu.Unlock()
	out := make([]string, 0, len(modReg.modules))
	for _, m := range modReg.modules {
		out = append(out, m.Name())
	}
	sort.Strings(out)
	return out
}

// ResetForTest mengosongkan registry (hanya untuk test).
func ResetForTest() {
	modReg.mu.Lock()
	modReg.modules = nil
	modReg.mu.Unlock()
	reset()
}
