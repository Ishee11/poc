package main

import (
	"log"

	"github.com/ishee11/poc/config"
	"github.com/ishee11/poc/internal/app"
)

func main() {
	// Configuration
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	// Run
	app.Run(cfg)
}
