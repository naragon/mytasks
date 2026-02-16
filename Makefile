.PHONY: build test run clean docker-build docker-run

# Binary name
BINARY=mytasks

# Build the application
build:
	go build -o $(BINARY) .

# Run tests
test:
	go test ./... -v

# Run tests with coverage
test-coverage:
	go test ./... -v -cover -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# Run the application locally
run: build
	./$(BINARY)

# Run with custom port
run-dev:
	PORT=3000 DB_PATH=./data/dev.db go run .

# Clean build artifacts
clean:
	rm -f $(BINARY)
	rm -f coverage.out coverage.html
	rm -rf ./data

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	go vet ./...

# Tidy dependencies
tidy:
	go mod tidy

# Build Docker image
docker-build:
	docker build -t $(BINARY):latest .

# Run Docker container
docker-run:
	docker run -p 8080:8080 -v $$(pwd)/data:/data $(BINARY):latest

# Run Docker container in detached mode
docker-run-detached:
	docker run -d -p 8080:8080 -v $$(pwd)/data:/data --name $(BINARY) $(BINARY):latest

# Stop and remove Docker container
docker-stop:
	docker stop $(BINARY) || true
	docker rm $(BINARY) || true

# Full rebuild of Docker image
docker-rebuild: docker-stop
	docker build --no-cache -t $(BINARY):latest .

# All-in-one development setup
dev: tidy fmt lint test run-dev

# All-in-one production build
prod: tidy fmt lint test build
