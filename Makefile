.PHONY:
.SILENT:

build:
	docker-compose up --build

stop:
	docker-compose stop

spam:
	go run cmd/spamer/spamer.go

stop-spam:
	./kill.sh
