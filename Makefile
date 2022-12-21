.PHONY:
.SILENT:


run:
	docker-compose up -d

build:
	docker-compose up -d --build

stop:
	docker-compose stop

spam:
	go run cmd/spamer/main.go
