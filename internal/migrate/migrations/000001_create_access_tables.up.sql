-- Skema modul access (RBAC). SQL portabel (VARCHAR/TEXT/BIGINT/BOOLEAN/TIMESTAMP)
-- → jalan di MySQL, Postgres, SQLite. golang-migrate menjamin sekali-jalan
-- (tak perlu IF NOT EXISTS pada index — MySQL menolaknya).

-- Skema KANONIK lintas-port (1:1 NodeAdmin): permissions.guard_name (web/api, untuk filter),
-- name varchar(255) (permissions.name NON-unik, roles.name unik), created_by/updated_by.
CREATE TABLE IF NOT EXISTS permissions (
  id varchar(36) PRIMARY KEY,
  name varchar(255) NOT NULL,
  guard_name varchar(20) DEFAULT 'web',
  created_by varchar(36),
  updated_by varchar(36),
  created_at timestamp DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX ix_permissions_name ON permissions (name);
CREATE INDEX ix_permissions_guard ON permissions (guard_name);

CREATE TABLE IF NOT EXISTS roles (
  id varchar(36) PRIMARY KEY,
  name varchar(255) NOT NULL,
  created_by varchar(36),
  updated_by varchar(36),
  created_at timestamp DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX ux_roles_name ON roles (name);

CREATE TABLE IF NOT EXISTS users (
  id varchar(36) PRIMARY KEY,
  code varchar(20),
  name varchar(50),
  phone varchar(15),
  email varchar(255) NOT NULL,
  email_verified_at timestamp NULL,
  password varchar(255),
  password_otp varchar(255),
  password_otp_expires bigint,
  status varchar(20) DEFAULT 'Active',
  picture varchar(255),
  blocked boolean DEFAULT FALSE,
  blocked_reason varchar(255),
  timezone varchar(64) DEFAULT 'UTC',
  created_by varchar(36),
  updated_by varchar(36),
  created_at timestamp DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX ux_users_code ON users (code);
CREATE UNIQUE INDEX ux_users_email ON users (email);
CREATE INDEX ix_users_name ON users (name);
CREATE INDEX ix_users_phone ON users (phone);
CREATE INDEX ix_users_status ON users (status);
CREATE INDEX ix_users_blocked ON users (blocked);
CREATE INDEX ix_users_timezone ON users (timezone);

CREATE TABLE IF NOT EXISTS roles_permissions (
  role_id varchar(36) NOT NULL,
  permission_id varchar(36) NOT NULL,
  PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS users_roles (
  user_id varchar(36) NOT NULL,
  role_id varchar(36) NOT NULL,
  PRIMARY KEY (user_id, role_id)
);
