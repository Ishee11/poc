// @title Poker API
// @version 1.0
// @description Poker sessions service

// @BasePath /
package main

import (
	"log"

	"github.com/ishee11/poc/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
