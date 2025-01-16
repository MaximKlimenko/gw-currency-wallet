# Variables
APP_NAME = gw-currency-wallet
CONFIG_FILE = config.env

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	GOOS=linux GOARCH=amd64 go build -o main ./cmd

# Run the application locally
.PHONY: run
run:
	./main -c $(CONFIG_FILE)

# Build the Docker image
.PHONY: docker-build
docker-build:
	docker build -t $(APP_NAME) .

# Run the Docker container
.PHONY: docker-run
docker-run:
	docker run -p 3000:3000 --env-file=$(CONFIG_FILE) $(APP_NAME)

# Clean up build files
.PHONY: clean
clean:
	rm -f main