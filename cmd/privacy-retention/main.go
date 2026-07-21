package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres"
	"gorm.io/gorm"
)

type retentionRule struct {
	name   string
	table  string
	column string
	days   int
}

func main() {
	apply := flag.Bool("apply", false, "delete records; without this flag only reports counts")
	flag.Parse()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	db, err := postgres.Open(databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	rules := []retentionRule{
		{name: "whatsapp recipients", table: "message_recipients", column: "created_at", days: envDays("PRIVACY_RETENTION_COMMUNICATION_DAYS", 365)},
		{name: "email recipients", table: "email_recipients", column: "created_at", days: envDays("PRIVACY_RETENTION_COMMUNICATION_DAYS", 365)},
		{name: "workout recipients", table: "workout_message_recipients", column: "created_at", days: envDays("PRIVACY_RETENTION_COMMUNICATION_DAYS", 365)},
		{name: "LLM logs", table: "llm_generation_logs", column: "created_at", days: envDays("PRIVACY_RETENTION_LLM_LOG_DAYS", 90)},
		{name: "automation runs", table: "automation_runs", column: "started_at", days: envDays("PRIVACY_RETENTION_AUTOMATION_RUN_DAYS", 180)},
		{name: "imports and checkins", table: "import_histories", column: "imported_at", days: envDays("PRIVACY_RETENTION_CHECKIN_DAYS", 730)},
		{name: "privacy audit", table: "privacy_audit_events", column: "created_at", days: envDays("PRIVACY_RETENTION_AUDIT_DAYS", 1825)},
	}
	mode := "dry-run"
	if *apply {
		mode = "apply"
	}
	fmt.Printf("privacy retention mode=%s\n", mode)
	err = db.WithContext(context.Background()).Transaction(func(tx *gorm.DB) error {
		for _, rule := range rules {
			cutoff := time.Now().UTC().AddDate(0, 0, -rule.days)
			var count int64
			if err := tx.Table(rule.table).Where(rule.column+" < ?", cutoff).Count(&count).Error; err != nil {
				return err
			}
			fmt.Printf("%s: %d eligible (older than %d days)\n", rule.name, count, rule.days)
			if *apply && count > 0 {
				if err := tx.Exec("DELETE FROM "+rule.table+" WHERE "+rule.column+" < ?", cutoff).Error; err != nil {
					return err
				}
			}
		}
		if !*apply {
			return gorm.ErrInvalidTransaction
		}
		return nil
	})
	if !*apply && err == gorm.ErrInvalidTransaction {
		err = nil
	}
	if err != nil {
		log.Fatal(err)
	}
}

func envDays(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	days, err := strconv.Atoi(value)
	if err != nil || days < 1 {
		log.Fatalf("%s must be a positive integer", name)
	}
	return days
}
