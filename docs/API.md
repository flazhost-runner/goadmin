# API GoAdmin (REST)

Base path: **`/api/v1`**. Tersedia di kedua varian (`full` & `api`).

> **Postman:** impor [`docs/postman/GoAdmin.postman_collection.json`](postman/GoAdmin.postman_collection.json) untuk mencoba semua endpoint. Variabel koleksi `base_url` default `http://localhost:3000` (sesuai `APP_PORT` default) — sesuaikan bila server jalan di port lain.

## Autentikasi

JWT **HS256** via header `Authorization: Bearer <token>`. Token didapat dari login, dicabut (blacklist) saat logout. Endpoint ber-RBAC butuh permission spesifik; **Administrator bypass** semua permission.

## Bentuk respons

```jsonc
// sukses
{ "success": true,  "message": "OK", "data": { /* ... */ } }
// gagal (status sesuai AppError: 400/401/403/404/409/422/429/500)
{ "success": false, "message": "Email atau password salah", "errors": { /* opsional per-field */ } }
```

Paginasi (list): query `page`, `per_page`, `search`; `data` berisi `{ data: [...], meta: { total, per_page, current_page, last_page, from, to } }`.

---

## Auth

| Method | Path | Auth | Keterangan |
|---|---|---|---|
| POST | `/api/v1/auth/login` | publik | body `{ email, password }` → `{ token, user }` |
| POST | `/api/v1/auth/register` | publik (rate-limit) | `{ name, email, password }` → user dibuat |
| POST | `/api/v1/auth/reset/request` | publik (rate-limit) | `{ email }` → kirim OTP (anti user-enumeration) |
| POST | `/api/v1/auth/reset/process` | publik (rate-limit) | `{ email, otp, password }` → set password baru |
| POST | `/api/v1/auth/logout` | JWT | cabut token saat ini (blacklist) — POST (mutasi) |
| GET | `/api/v1/auth/me` | JWT | profil user dari token |

## Users — RBAC `user.*`

| Method | Path | Permission | Body |
|---|---|---|---|
| GET | `/api/v1/access/user` | `user.view` | — (query: page/per_page/search) |
| POST | `/api/v1/access/user/store` | `user.create` | `{ name, email, phone?, password, status?, timezone?, role_ids?[] }` |
| GET | `/api/v1/access/user/:id/edit` | `user.view` | — (entity utk edit) |
| PUT | `/api/v1/access/user/:id/update` | `user.update` | sama, `password` opsional |
| DELETE | `/api/v1/access/user/:id/delete` | `user.delete` | — |
| POST | `/api/v1/access/user/delete_selected` | `user.delete` | `{ selected: [id,...] }` |

## Roles — RBAC `role.*`

| Method | Path | Permission | Body |
|---|---|---|---|
| GET | `/api/v1/access/role` | `role.view` | — |
| POST | `/api/v1/access/role/store` | `role.create` | `{ name, permission_ids?[] }` |
| GET | `/api/v1/access/role/:id/edit` | `role.view` | — (entity utk edit) |
| PUT | `/api/v1/access/role/:id/update` | `role.update` | `{ name, permission_ids?[] }` |
| DELETE | `/api/v1/access/role/:id/delete` | `role.delete` | — (role Administrator ditolak) |
| POST | `/api/v1/access/role/delete_selected` | `role.delete` | `{ selected: [id,...] }` |
| GET | `/api/v1/access/role/:id/permission` | (route-driven) | — daftar permission + status assigned (filter `q_name`/`q_status`/`q_desc`) |
| GET | `/api/v1/access/role/:id/permission/:permission_id/assign` | (route-driven) | — assign 1 permission |
| POST | `/api/v1/access/role/:id/permission/assign_selected` | (route-driven) | `{ selected: [id,...] }` |
| GET | `/api/v1/access/role/:id/permission/:permission_id/unassign` | (route-driven) | — unassign 1 permission |
| POST | `/api/v1/access/role/:id/permission/unassign_selected` | (route-driven) | `{ selected: [id,...] }` |

## Permissions — RBAC `permission.*`

| Method | Path | Permission | Body |
|---|---|---|---|
| GET | `/api/v1/access/permission` | `permission.view` | — |
| POST | `/api/v1/access/permission/store` | `permission.create` | `{ name }` |
| GET | `/api/v1/access/permission/:id/edit` | `permission.view` | — (entity utk edit) |
| PUT | `/api/v1/access/permission/:id/update` | `permission.update` | `{ name }` |
| DELETE | `/api/v1/access/permission/:id/delete` | `permission.delete` | — |
| POST | `/api/v1/access/permission/delete_selected` | `permission.delete` | `{ selected: [id,...] }` |

## Setting — RBAC `setting.*`

| Method | Path | Permission | Body |
|---|---|---|---|
| GET | `/api/v1/setting` | `setting.view` | — → `{ setting, themes }` |
| PUT | `/api/v1/setting/update` | `setting.update` | `{ name?, initial?, description?, phone?, address?, email?, copyright?, theme?, fe_template? }` (parsial) |

## Profile — JWT (tanpa permission khusus)

| Method | Path | Auth | Body |
|---|---|---|---|
| GET | `/api/v1/profile` | JWT | — |
| PUT | `/api/v1/profile/update` | JWT | `{ name, email, phone?, timezone?, password?, password_confirmation? }` |

> Profil least-privilege: tak bisa mengubah status/role sendiri.

## Dashboard — JWT

| Method | Path | Auth | Keterangan |
|---|---|---|---|
| GET | `/api/v1/dashboard` | JWT | `{ users, roles, permissions }` |

## Lain-lain

| Method | Path | Keterangan |
|---|---|---|
| GET | `/healthz` | health check (publik) → `{ status: "ok" }` |

---

### Contoh

```bash
# Login
curl -s -X POST http://localhost:3000/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@admin.com","password":"12345678"}'

# Pakai token
TOKEN=... # dari data.token di atas
curl -s http://localhost:3000/api/v1/access/user -H "Authorization: Bearer $TOKEN"
```
