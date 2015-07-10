.PHONY: serve test db-reset deps watch-serve docker-deps docker-build docker-run

serve:
	go install github.com/samertm/syncfbevents
	syncfbevents

watch-serve:
	$(shell while true; do $(MAKE) serve & PID=$$! ; echo $$PID ; inotifywait --exclude ".git" -r -e close_write . ; kill $$PID ; done)

deps:
	echo "Make sure you set up Postgres correctly."

db-reset:
	psql -d syncfbevents -c "drop schema public cascade"
	psql -d syncfbevents -c "create schema public"

test:
	go test ./...

docker-deps:
	docker pull postgres
	docker run --name sfe-db -d postgres
	sleep 5 # Wait for database to be created.
	docker exec sfe-db psql -U postgres -c 'CREATE USER sfe'
	docker exec sfe-db psql -U postgres -c 'CREATE DATABASE sfe'
	docker exec sfe-db psql -U postgres -c 'GRANT ALL PRIVILEGES ON DATABASE sfe TO sfe'
	docker stop sfe-db
	docker start sfe-db

docker-build:
	docker build -t sfe .

docker-run:
	docker start sfe-db # Did you run 'make docker-deps'?
	-docker top sfe-app && docker stop sfe-app && docker rm sfe-app
	docker run -d -p 8111:8000 --name sfe-app --link sfe-db:postgres sfe # Did you run 'make docker-build?'

deploy-deps:
	rsync -azP . samertm:~/syncfbevents
	ssh $(TO) 'cd ~/syncfbevents && make docker-deps'

deploy:
	rsync -azP . samertm:~/syncfbevents
	ssh $(TO) 'cd ~/syncfbevents && make docker-build && make docker-run'
