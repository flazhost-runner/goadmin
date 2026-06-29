-- Skema modul setting (singleton konfigurasi global).
CREATE TABLE IF NOT EXISTS settings (
  id varchar(36) PRIMARY KEY,
  initial varchar(255),
  name varchar(255),
  description text,
  icon varchar(255),
  logo varchar(255),
  favicon varchar(255),
  login_image varchar(255),
  phone varchar(255),
  address varchar(255),
  email varchar(255),
  copyright varchar(255),
  theme varchar(20) DEFAULT 'Blue',
  fe_template varchar(80),
  created_by varchar(36),
  updated_by varchar(36),
  created_at timestamp DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX ix_settings_initial ON settings (initial);
CREATE INDEX ix_settings_name ON settings (name);
CREATE INDEX ix_settings_email ON settings (email);
