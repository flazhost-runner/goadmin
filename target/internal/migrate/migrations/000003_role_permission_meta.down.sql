-- Reversibel: DROP COLUMN ikut membuang index terkait (MySQL & Postgres).
-- `desc` di-quote (ANSI double-quote; MySQL DSN migrate = ANSI_QUOTES).

ALTER TABLE permissions DROP COLUMN "desc";
ALTER TABLE permissions DROP COLUMN status;
ALTER TABLE permissions DROP COLUMN method;

ALTER TABLE roles DROP COLUMN "desc";
ALTER TABLE roles DROP COLUMN status;
ALTER TABLE roles DROP COLUMN guard_name;
