package database

import (
	"testing"

	"goadmin/internal/config"
)

// Batas pool berlaku PER REPLIKA, sementara kuota DB terkelola dibagi ke SEMUA replika.
// Karena itu konfigurasinya harus jujur: idle tak boleh melebihi open, dan limit 0 tak
// boleh diam-diam berarti "tak terbatas".
func TestPoolLimits(t *testing.T) {
	cases := []struct {
		name     string
		cfg      config.DBConfig
		wantOpen int
		wantIdle int
	}{
		{
			name:     "tier kecil: idle dipangkas agar tak melebihi limit",
			cfg:      config.DBConfig{ConnMaxOpen: 2, ConnMaxIdle: 5},
			wantOpen: 2,
			wantIdle: 2,
		},
		{
			name:     "limit 0 tidak berarti tak terbatas",
			cfg:      config.DBConfig{ConnMaxOpen: 0, ConnMaxIdle: 5},
			wantOpen: defaultMaxOpen,
			wantIdle: 5,
		},
		{
			name:     "idle negatif dinormalkan ke 0",
			cfg:      config.DBConfig{ConnMaxOpen: 10, ConnMaxIdle: -3},
			wantOpen: 10,
			wantIdle: 0,
		},
		{
			name:     "konfigurasi wajar dibiarkan apa adanya",
			cfg:      config.DBConfig{ConnMaxOpen: 10, ConnMaxIdle: 5},
			wantOpen: 10,
			wantIdle: 5,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			open, idle := poolLimits(tc.cfg)
			if open != tc.wantOpen || idle != tc.wantIdle {
				t.Errorf("poolLimits() = (open=%d, idle=%d), mau (open=%d, idle=%d)",
					open, idle, tc.wantOpen, tc.wantIdle)
			}
		})
	}
}
