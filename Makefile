# Makefile GoAdmin — entrypoint task lokal & CI.
# Padanan script npm di NodeAdmin (lint:conventions, test, dll).

.PHONY: check lint test build run vet tidy verify module migrate migration

# module = scaffold modul CRUD baru (otomatis ikut pola `access` + lolos checker).
# Contoh:
#   make module ARGS="--name product"            # full (api + web)
#   make module ARGS="--name token --web=false"  # api-only
#   make module ARGS="--name category --plural categories"
module:
	@go run ./cmd/make-module $(ARGS)

# check = convention checker (guardrail SOLID/DI, error handling, portabilitas
# DB, security). Gate: exit 1 bila ada pelanggaran. Padanan `npm run lint:conventions`.
check:
	@go run ./cmd/checkconventions

# vet = analisa statis bawaan Go.
vet:
	@go vet ./...

# test = seluruh unit/integration/api test.
test:
	@go test ./...

# build = kompilasi semua paket (deteksi error kompilasi dini).
build:
	@go build ./...

# migrate = migrasi DB. sqlite → AutoMigrate (dev); mysql/postgres → golang-migrate
# versioned (.up/.down SQL). Contoh:
#   DB_TYPE=sqlite DB_DATABASE=goadmin.db make migrate     # up + seed
#   make migrate ARGS="-down 1"     # rollback (mysql/postgres)
#   make migrate ARGS="-version"    # versi saat ini
migrate:
	@go run ./cmd/migrate $(ARGS)

# migration = buat pasangan file migrasi baru. Contoh: make migration name=add_orders
migration:
	@go run ./cmd/migrate -create "$(name)"

# run = jalankan server.
run:
	@go run ./cmd/server

# tidy = rapikan go.mod/go.sum.
tidy:
	@go mod tidy

# verify = gate lengkap sebelum dianggap selesai (urutan: konvensi → vet →
# build → test). Pakai ini di CI / sebelum commit.
verify: check vet build test
	@echo "\n✅ verify: konvensi + vet + build + test lolos."
