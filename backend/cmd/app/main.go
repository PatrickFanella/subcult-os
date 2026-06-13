package main

import (
	"context"
	"log"
	"net/http"

	"git.subcult.tv/PatrickFanella/subcult-os/internal/app"
)

func main() {
	ctx := context.Background()
	config := app.LoadConfig()
	if err := config.Validate(); err != nil {
		log.Fatal(err)
	}
	db, err := app.OpenDB(ctx, config.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	if db != nil {
		defer db.Close()
	}
	if err := app.RunMigrations(ctx, db); err != nil {
		log.Fatal(err)
	}

	server := app.New(config, db)
	log.Printf("subcult-os api listening on %s", config.Addr)
	log.Fatal(http.ListenAndServe(config.Addr, server.Handler()))
}
