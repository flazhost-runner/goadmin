// Package helpers berisi util DRY yang dipakai lintas modul (padanan
// src/helpers/functions.ts di NodeAdmin): pagination, pencarian
// case-insensitive portabel (ciLike), dan pembersihan field kosong.
package helpers

import (
	"strings"

	"gorm.io/gorm"
)

// CiLike menerapkan pencarian case-insensitive yang PORTABEL lintas dialek.
//
// Mengapa tidak `LIKE ?` langsung: MySQL default case-insensitive, tapi
// PostgreSQL & SQLite case-sensitive untuk LIKE. Dengan membungkus kedua sisi
// LOWER() perilakunya seragam di semua dialek (pelajaran NodeAdmin ciLike()).
// Checker menolak `LIKE` manual di modul → wajib lewat helper ini.
func CiLike(db *gorm.DB, column, keyword string) *gorm.DB {
	kw := strings.TrimSpace(keyword)
	if kw == "" {
		return db
	}
	// Quote identifier per-dialek (aman utk reserved word seperti `desc`).
	return db.Where("LOWER("+db.Statement.Quote(column)+") LIKE LOWER(?)", "%"+kw+"%")
}

// CiLikeAny mencari keyword di banyak kolom sekaligus (OR), tetap portabel.
func CiLikeAny(db *gorm.DB, columns []string, keyword string) *gorm.DB {
	kw := strings.TrimSpace(keyword)
	if kw == "" || len(columns) == 0 {
		return db
	}
	conds := make([]string, 0, len(columns))
	args := make([]interface{}, 0, len(columns))
	for _, c := range columns {
		conds = append(conds, "LOWER("+db.Statement.Quote(c)+") LIKE LOWER(?)")
		args = append(args, "%"+kw+"%")
	}
	return db.Where(strings.Join(conds, " OR "), args...)
}
