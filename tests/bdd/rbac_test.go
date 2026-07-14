// Package bdd menjalankan skenario perilaku (Gherkin) dengan cucumber/godog
// terhadap aplikasi API NYATA (app.Build + httptest + SQLite in-memory).
package bdd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/cucumber/godog"
	"github.com/gin-gonic/gin"

	"goadmin/internal/app"
	"goadmin/internal/bootstrap"
	"goadmin/internal/config"
	"goadmin/internal/container"
	"goadmin/internal/database"
	accessdto "goadmin/internal/modules/access/dto"
	accesssvc "goadmin/internal/modules/access/service"

	_ "goadmin/internal/modules" // registrasi modul (side-effect)
)

const adminEmail, adminPass = "admin@bdd.test", "secret123"

var dbSeq int64

// world = state satu skenario (engine API + token terakhir + status terakhir).
type world struct {
	engine *gin.Engine
	cont   *container.Container
	token  string
	status int
}

func newWorld() (*world, error) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		Env: "test", IsTest: true,
		App:      config.AppConfig{Name: "GoAdmin BDD", Mode: config.ModeAPI},
		DB:       config.DBConfig{Type: "sqlite"},
		JWT:      config.JWTConfig{Secret: "bdd-secret", Algorithm: "HS256"},
		Session:  config.SessionConfig{Secret: "bdd-session"},
		Security: config.SecurityConfig{BcryptRounds: 4},
	}
	name := fmt.Sprintf("bdd_%d", atomic.AddInt64(&dbSeq, 1))
	db, err := database.OpenSQLiteMemory(name)
	if err != nil {
		return nil, err
	}
	if err := bootstrap.MigrateAndSeed(db, adminEmail, adminPass, cfg.Security.BcryptRounds); err != nil {
		return nil, err
	}
	c := container.MustNew(cfg, db, nil)
	return &world{engine: app.Build(c), cont: c}, nil
}

func (w *world) login(email, password string) error {
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w.engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		return fmt.Errorf("login gagal (%d): %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		return err
	}
	if resp.Data.Token == "" {
		return fmt.Errorf("token kosong")
	}
	w.token = resp.Data.Token
	return nil
}

func (w *world) request(method, path string, withToken bool) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(strings.ToUpper(method), path, nil)
	if withToken {
		req.Header.Set("Authorization", "Bearer "+w.token)
	}
	w.engine.ServeHTTP(rec, req)
	w.status = rec.Code
}

// --- step definitions ---

func InitializeScenario(ctx *godog.ScenarioContext) {
	w := &world{}

	ctx.Step(`^aplikasi API siap dengan admin ter-seed$`, func() error {
		nw, err := newWorld()
		if err != nil {
			return err
		}
		*w = *nw
		return nil
	})

	ctx.Step(`^ada user biasa "([^"]*)" dengan password "([^"]*)"$`, func(email, pass string) error {
		svc := accesssvc.NewUserService(w.cont.DB, w.cont.Config.Security.BcryptRounds)
		_, err := svc.Store(context.Background(), accessdto.CreateUserInput{
			Name: "Plain", Email: email, Password: pass,
		}, "")
		return err
	})

	ctx.Step(`^admin login$`, func() error { return w.login(adminEmail, adminPass) })
	ctx.Step(`^user "([^"]*)" dengan password "([^"]*)" login$`, func(email, pass string) error {
		return w.login(email, pass)
	})

	ctx.Step(`^klien mengakses "([^"]*)" "([^"]*)" tanpa token$`, func(m, p string) error {
		w.request(m, p, false)
		return nil
	})
	ctx.Step(`^klien mengakses "([^"]*)" "([^"]*)" dengan token$`, func(m, p string) error {
		w.request(m, p, true)
		return nil
	})

	ctx.Step(`^status respons adalah (\d+)$`, func(code int) error {
		if w.status != code {
			return fmt.Errorf("harap status %d, dapat %d", code, w.status)
		}
		return nil
	})
}

// TestFeatures menjalankan seluruh skenario Gherkin di folder features/.
func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("ada skenario BDD yang gagal")
	}
}
