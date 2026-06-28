.PHONY: build clean

build:
	@echo "Building Lambda..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
	go build -tags lambda.norpc -o bootstrap ./cmd/lambda

	zip -j function.zip bootstrap

	rm bootstrap

clean:
	rm -f function.zip


