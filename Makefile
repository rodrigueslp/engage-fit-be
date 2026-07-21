COMPOSE=docker compose
DATABASE_URL=postgres://boxengage:boxengage@localhost:5432/boxengage?sslmode=disable

.PHONY: up down logs ps migrate-up migrate-status migrate-baseline rotate-secrets privacy-retention-dry-run privacy-retention-apply backend-run backend-test demo-seed demo-reset demo-reset-seed daily-automation observability-up observability-down observability-logs

up:
	$(COMPOSE) up -d postgres

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f

ps:
	$(COMPOSE) ps

migrate-up:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate up

migrate-status:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate status

# Somente para adotar um banco preexistente ja verificado. Ex.: make migrate-baseline VERSION=30
migrate-baseline:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate --through=$(VERSION) baseline

rotate-secrets:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/rotate-secrets

privacy-retention-dry-run:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/privacy-retention

privacy-retention-apply:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/privacy-retention --apply

backend-run:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/api

backend-test:
	go test ./...

demo-seed:
	API_BASE_URL=http://localhost:8080 node scripts/demo-seed.mjs

demo-reset:
	$(COMPOSE) exec -T postgres psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -c "CREATE UNIQUE INDEX IF NOT EXISTS idx_checkins_unique_visit ON checkins (box_id, source, student_id, checkin_date, checkin_time);"
	$(COMPOSE) exec -T postgres psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -c "BEGIN; TRUNCATE TABLE admin_audit_logs, messaging_usage_buckets RESTART IDENTITY CASCADE; DELETE FROM boxes; DELETE FROM messaging_policies WHERE scope = 'platform'; COMMIT;"

demo-reset-seed: demo-reset demo-seed

daily-automation:
	API_BASE_URL=http://localhost:8080 node scripts/daily-automation.mjs

observability-up:
	$(COMPOSE) -f docker-compose.observability.yml up -d

observability-down:
	$(COMPOSE) -f docker-compose.observability.yml down

observability-logs:
	$(COMPOSE) -f docker-compose.observability.yml logs -f
