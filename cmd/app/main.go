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
