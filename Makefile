.PHONY: build test clean docker-build docker-push deploy run

# Variables
SERVICE_NAME=$(shell basename $(CURDIR))
GCP_PROJECT_ID?=ecommerce-platform-475712
TAG?=latest
IMAGE_NAME=gcr.io/$(GCP_PROJECT_ID)/$(SERVICE_NAME):$(TAG)

# Go commands
build:
	go build -o $(SERVICE_NAME) ./cmd/server/main.go

test:
	go test -v ./...

clean:
	rm -f $(SERVICE_NAME)
	go clean

# Docker commands
docker-build:
	docker build -t $(IMAGE_NAME) .

docker-push: docker-build
	docker push $(IMAGE_NAME)

# Deployment commands
deploy: docker-push
	kubectl set image deployment/$(SERVICE_NAME) $(SERVICE_NAME)=$(IMAGE_NAME) -n ecommerce-staging
	kubectl rollout status deployment/$(SERVICE_NAME) -n ecommerce-staging

# Local development
run:
	go run ./cmd/server/main.go

# Dependencies setup
setup-deps:
	go mod tidy
	go mod verify