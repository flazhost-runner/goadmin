// Package service berisi logika landing publik (home). Membangun view-model
// landing dari Setting global (ber-cache) dengan fallback aman. Mengembalikan
// *errors.AppError saat gagal & mengimplementasi IHomeService.
package service

import "context"

// LandingData = view-model landing publik (hasil binding Setting + fallback).
// Sengaja datar (bukan entity) agar view tak bergantung struktur DB.
type LandingData struct {
	AppName     string `json:"app_name"`
	Description string `json:"description"`
	Logo        string `json:"logo"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Address     string `json:"address"`
	Copyright   string `json:"copyright"`
	// Tema aktif (dipakai mewarnai landing — ikat ke theme switcher admin).
	ThemeName string `json:"theme_name"`
	Primary   string `json:"primary"`
	Accent    string `json:"accent"`
	// Template = slug landing aktif (frontend template switcher) → view "home/<slug>".
	Template string `json:"template"`
}

// IHomeService = kontrak data landing publik.
type IHomeService interface {
	// Landing membangun view-model landing dari Setting (dengan fallback).
	Landing(ctx context.Context) (LandingData, error)
}
