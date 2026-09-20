package main

import (
	"log"

	"github.com/GaM1rka/url-shortener/internal/app"
	"github.com/GaM1rka/url-shortener/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	if err := application.Run(); err != nil {
		log.Fatal(err)
	}
}