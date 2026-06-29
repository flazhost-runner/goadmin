-- Tambah metadata Role & Permission agar 1:1 dengan NodeAdmin:
--   roles       : status (default Active), desc
--   permissions : method, status (default Active), desc
-- Kolom `desc` = RESERVED WORD → di-quote double-quote (ANSI). Portabel di
-- Postgres (native) & MySQL (DSN migrate pakai sql_mode=ANSI_QUOTES). SQLite
-- dev memakai AutoMigrate (model GORM, column:desc) — bukan file ini.

ALTER TABLE roles ADD COLUMN guard_name varchar(50) DEFAULT 'web';
ALTER TABLE roles ADD COLUMN status varchar(20) DEFAULT 'Active';
ALTER TABLE roles ADD COLUMN "desc" varchar(255);
CREATE INDEX ix_roles_guard ON roles (guard_name);
CREATE INDEX ix_roles_status ON roles (status);
CREATE INDEX ix_roles_desc ON roles ("desc");

ALTER TABLE permissions ADD COLUMN method varchar(255);
ALTER TABLE permissions ADD COLUMN status varchar(20) DEFAULT 'Active';
ALTER TABLE permissions ADD COLUMN "desc" varchar(255);
CREATE INDEX ix_permissions_method ON permissions (method);
CREATE INDEX ix_permissions_status ON permissions (status);
CREATE INDEX ix_permissions_desc ON permissions ("desc");
