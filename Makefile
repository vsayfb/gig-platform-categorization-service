.PHONY: build run clean

build:
	@echo "Building Go binary for AWS Lambda (AL2023)..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
	go build -tags lambda.norpc -o bootstrap ./cmd/server

	@echo "Packaging Lambda..."
	zip -j function.zip bootstrap

	@rm bootstrap

run: build
	@echo "Starting LocalStack..."
	docker compose up

clean:
	@echo "Cleaning build artifacts..."
	rm -f function.zip