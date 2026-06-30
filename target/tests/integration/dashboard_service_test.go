package integration

import (
	"context"
	"testing"

	"goadmin/internal/config"
	"goadmin/internal/helpers"
	"goadmin/internal/modules/access/model"
	"goadmin/internal/modules/dashboard/service"
	"goadmin/tests/testutil"
)

// DB kosong (skema access ada, tanpa seed) → semua hitungan nol.
func TestDashboardService_StatsEmpty(t *testing.T) {
	c := testutil.NewContainer(t, config.ModeFull)
	svc := service.NewDashboardService(c.DB)

	st, err := svc.Stats(context.Background())
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if st.Users != 0 || st.Roles != 0 || st.Permissions != 0 {
		t.Fatalf("harusnya nol, dapat: %+v", st)
	}
}

// Setelah seed admin: 1 user, 1 role Administrator, 0 permission. Permission kini
// ROUTE-DRIVEN (di-sync dari registry route, a la NodeAdmin) — TIDAK lagi di-seed.
// Lalu buat permission manual → Stats menghitung dari tabel.
func TestDashboardService_StatsAfterSeed(t *testing.T) {
	c := testutil.NewContainer(t, config.ModeFull)
	testutil.SeedAdmin(t, c)
	svc := service.NewDashboardService(c.DB)

	st, err := svc.Stats(context.Background())
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if st.Users != 1 {
		t.Fatalf("users: harap 1, dapat %d", st.Users)
	}
	if st.Roles != 1 {
		t.Fatalf("roles: harap 1, dapat %d", st.Roles)
	}
	if st.Permissions != 0 {
		t.Fatalf("permissions: harap 0 (route-driven, tak di-seed), dapat %d", st.Permissions)
	}

	// Permission diturunkan dari route (di sini disimulasikan manual) → terhitung.
	for _, p := range []model.Permission{
		{ID: helpers.NewID(), Name: "admin.v1.access.user.index", Method: "GET", GuardName: "web", Status: model.StatusActive},
		{ID: helpers.NewID(), Name: "admin.v1.access.user.delete", Method: "DELETE", GuardName: "web", Status: model.StatusActive},
	} {
		if err := c.DB.Create(&p).Error; err != nil {
			t.Fatalf("create perm: %v", err)
		}
	}
	st2, err := svc.Stats(context.Background())
	if err != nil {
		t.Fatalf("stats2: %v", err)
	}
	if st2.Permissions != 2 {
		t.Fatalf("permissions: harap 2, dapat %d", st2.Permissions)
	}
}
