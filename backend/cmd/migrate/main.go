package main

import (
	"context"
	"log"

	"git.subcult.tv/PatrickFanella/subcult-os/internal/app"
)

func main() {
	ctx := context.Background()
	config := app.LoadConfig()
	if config.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := app.OpenDB(ctx, config.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := app.RunMigrations(ctx, db); err != nil {
		log.Fatal(err)
	}
	version, err := app.CurrentSchemaVersion(ctx, db)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("subcult-os schema is at version %d", version)
}
