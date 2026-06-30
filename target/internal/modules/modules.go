// Package modules adalah agregator registrasi modul. Mengimpor tiap modul
// sebagai side-effect (tiap modul memanggil router.Add di init-nya), sehingga
// main cukup `import _ "goadmin/internal/modules"`.
//
// Modul UI-only didaftarkan dengan guard kehadiran di Register (bila ctx.Web
// nil pada mode api → lewati registrasi web), menjaga diff full↔api additive.
package modules

import (
	// Modul referensi (RBAC). Modul lain ditambahkan di bawah seiring fase.
	_ "goadmin/internal/modules/access"
	_ "goadmin/internal/modules/components"
	_ "goadmin/internal/modules/dashboard"
	_ "goadmin/internal/modules/home"
	_ "goadmin/internal/modules/media"
	_ "goadmin/internal/modules/profile"
	_ "goadmin/internal/modules/setting"
)
