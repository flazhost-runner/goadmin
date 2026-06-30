package helpers

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// fallbackTimezones = subset kurasi (PERSIS fallback NodeAdmin `getTimezones`)
// dipakai bila direktori tzdata sistem tak terbaca.
var fallbackTimezones = []string{
	"UTC", "Europe/London", "Europe/Berlin", "Europe/Paris", "Europe/Moscow",
	"Africa/Johannesburg", "Asia/Jakarta", "Asia/Bangkok", "Asia/Singapore",
	"Asia/Hong_Kong", "Asia/Shanghai", "Asia/Tokyo", "Asia/Seoul", "Asia/Kolkata",
	"Australia/Sydney", "Pacific/Auckland", "America/New_York", "America/Chicago",
	"America/Denver", "America/Los_Angeles", "America/Sao_Paulo",
}

// Timezones mengembalikan daftar nama zona waktu IANA (padanan
// Intl.supportedValuesOf('timeZone') NodeAdmin). Dibaca dari direktori tzdata
// sistem (zona "Region/City" + UTC); fallback ke subset kurasi NodeAdmin.
func Timezones() []string {
	for _, root := range []string{os.Getenv("ZONEINFO"), "/usr/share/zoneinfo", "/usr/share/lib/zoneinfo", "/etc/zoneinfo"} {
		if root == "" {
			continue
		}
		if zones := readZoneinfo(root); len(zones) > 0 {
			return zones
		}
	}
	return fallbackTimezones
}

func readZoneinfo(root string) []string {
	var zones []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			// Lewati pohon duplikat posix/right.
			if n := d.Name(); n == "posix" || n == "right" {
				return fs.SkipDir
			}
			return nil
		}
		rel, e := filepath.Rel(root, path)
		if e != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		// Hanya zona kanonik "Region/City" (punya '/') + UTC. Ini membuang file
		// metadata (zone.tab/leapseconds/+VERSION — semuanya di root, tanpa '/')
		// dan alias top-level legacy (CET/EST/GMT/…), mendekati daftar Intl.
		if rel == "UTC" || (strings.Contains(rel, "/") &&
			!strings.HasSuffix(rel, ".tab") && !strings.HasSuffix(rel, ".zi") && !strings.HasSuffix(rel, ".list")) {
			zones = append(zones, rel)
		}
		return nil
	})
	sort.Strings(zones)
	return zones
}
