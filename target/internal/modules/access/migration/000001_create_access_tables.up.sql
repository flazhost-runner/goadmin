-- Migration access (up). SQL portabel: tipe abstrak (VARCHAR/TIMESTAMP/BOOLEAN),
-- tanpa ENGINE=/AUTO_INCREMENT/backtick spesifik-vendor. Cocok MySQL/PG/SQLite.

CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    code VARCHAR(20) NOT NULL,
    name VARCHAR(50) NOT NULL,
    phone VARCHAR(15),
    email VARCHAR(255) NOT NULL,
    email_verified_at TIMESTAMP NULL,
    password VARCHAR(255) NOT NULL,
    password_otp VARCHAR(255),
    password_otp_expires BIGINT,
    status VARCHAR(20) DEFAULT 'Active',
    picture VARCHAR(255),
    blocked BOOLEAN DEFAULT FALSE,
    blocked_reason VARCHAR(255),
    timezone VARCHAR(64) DEFAULT 'UTC',
    created_by VARCHAR(36),
    updated_by VARCHAR(36),
    created_at TIMESTAMP NULL,
    updated_at TIMESTAMP NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS users__code ON users (code);
CREATE UNIQUE INDEX IF NOT EXISTS users__email ON users (email);
CREATE INDEX IF NOT EXISTS users__name ON users (name);
CREATE INDEX IF NOT EXISTS users__status ON users (status);

CREATE TABLE IF NOT EXISTS roles (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    guard_name VARCHAR(50),
    created_at TIMESTAMP NULL,
    updated_at TIMESTAMP NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS roles__name ON roles (name);

CREATE TABLE IF NOT EXISTS permissions (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    guard_name VARCHAR(50),
    created_at TIMESTAMP NULL,
    updated_at TIMESTAMP NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS permissions__name ON permissions (name);

CREATE TABLE IF NOT EXISTS users_roles (
    user_id VARCHAR(36) NOT NULL,
    role_id VARCHAR(36) NOT NULL,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE IF NOT EXISTS roles_permissions (
    role_id VARCHAR(36) NOT NULL,
    permission_id VARCHAR(36) NOT NULL,
    PRIMARY KEY (role_id, permission_id)
);
