package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"boxengage/backend/migrations"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	through := flag.Int64("through", 0, "ultima versao confirmada ao criar baseline")
	timeout := flag.Duration("timeout", 5*time.Minute, "timeout total da operacao")
	flag.Parse()
	if flag.NArg() != 1 {
		log.Fatal("uso: engagefit-migrate [--timeout=5m] [--through=N] <up|status|baseline>")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL deve ser configurada")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("falha ao conectar no PostgreSQL: %v", err)
	}

	migrator := migrations.New(db)
	switch flag.Arg(0) {
	case "up":
		executed, err := migrator.Up(ctx)
		if err != nil {
			log.Fatal(err)
		}
		for _, migration := range executed {
			fmt.Printf("applied %03d %s\n", migration.Version, migration.Name)
		}
		fmt.Printf("migration complete: %d applied\n", len(executed))
	case "status":
		statuses, err := migrator.Status(ctx)
		if err != nil {
			log.Fatal(err)
		}
		for _, status := range statuses {
			state := "pending"
			if status.Applied != nil {
				state = "applied"
			}
			fmt.Printf("%03d %-8s %s\n", status.Migration.Version, state, status.Migration.Name)
		}
	case "baseline":
		if *through == 0 {
			if value := os.Getenv("MIGRATION_BASELINE_VERSION"); value != "" {
				parsed, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					log.Fatalf("MIGRATION_BASELINE_VERSION invalida: %v", err)
				}
				*through = parsed
			}
		}
		marked, err := migrator.Baseline(ctx, *through)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("baseline complete: %d migrations marked through version %03d\n", len(marked), *through)
	default:
		log.Fatalf("acao desconhecida %q; use up, status ou baseline", flag.Arg(0))
	}
}
