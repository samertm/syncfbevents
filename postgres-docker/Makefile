.PHONY: docker-build run-dev

IMAGE_NAME=sfe-dbi
CONTAINER_NAME=sfe-db

docker-build:
	docker build -t $(IMAGE_NAME) .

run-dev:
	docker run -d -p 5432:5432 --name $(CONTAINER_NAME) $(IMAGE_NAME)

run-prod:
	docker run -d --name $(CONTAINER_NAME) $(IMAGE_NAME)
