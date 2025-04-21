package main

import (
	"log"

	"github.com/fajaragst/zoom-to-s3/internal/config"
	"github.com/fajaragst/zoom-to-s3/internal/server"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Config Error")
	}

	app := server.NewServer(cfg)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
