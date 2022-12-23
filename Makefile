.PHONY:
.SILENT:

build:
	docker-compose up --build  & ./stop-whole-service.sh & ./stop-random-target.sh
