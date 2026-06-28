COMPOSE=docker compose
DATABASE_URL=postgres://boxengage:boxengage@localhost:5432/boxengage?sslmode=disable

.PHONY: up down logs ps migrate-up backend-run backend-test demo-seed demo-reset demo-reset-seed daily-automation

up:
	$(COMPOSE) up -d postgres

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f

ps:
	$(COMPOSE) ps

migrate-up:
	@for file in migrations/*.sql; do \
		echo "applying $$file"; \
		$(COMPOSE) exec -T postgres psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 < $$file; \
	done

backend-run:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/api

backend-test:
	go test ./...

demo-seed:
	API_BASE_URL=http://localhost:8080 node scripts/demo-seed.mjs

demo-reset:
	$(COMPOSE) exec -T postgres psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -c "CREATE UNIQUE INDEX IF NOT EXISTS idx_checkins_unique_visit ON checkins (box_id, source, student_id, checkin_date, checkin_time);"
	$(COMPOSE) exec -T postgres psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -c "TRUNCATE TABLE message_recipients, message_campaigns, message_templates, reward_deliveries, rewards, campaign_progresses, campaign_goals, campaigns, checkins, import_histories, students RESTART IDENTITY CASCADE;"

demo-reset-seed: demo-reset demo-seed

daily-automation:
	API_BASE_URL=http://localhost:8080 node scripts/daily-automation.mjs
