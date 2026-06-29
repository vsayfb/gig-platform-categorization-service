package main

import (
	"context"
	"log"

	"github.com/vsayfb/gig-platform-categorization-service/internal/worker"
)

func main() {
	ctx := context.Background()

	app, err := getApp(ctx)
	if err != nil {
		log.Fatal(err)
	}

	w, err := worker.New(app)
	if err != nil {
		log.Fatal(err)
	}

	if err := w.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
