-- Migration access (down). Reversible: hapus tabel dalam urutan aman FK.
DROP TABLE IF EXISTS roles_permissions;
DROP TABLE IF EXISTS users_roles;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS users;
