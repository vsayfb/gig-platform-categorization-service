package main

import (
	"context"
	"log"
)

func main() {
	ctx := context.Background()

	app, err := getApp(ctx)
	if err != nil {
		log.Fatal(err)
	}

	worker := NewWorker(app)

	if err := worker.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
