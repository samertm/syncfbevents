.PHONY: serve watch-serve db-reset test docker-deps docker-build docker-run deploy-deps deploy docker

serve:
	go install github.com/samertm/syncfbevents
	syncfbevents

watch-serve:
	$(shell while true; do $(MAKE) serve & PID=$$! ; echo $$PID ; inotifywait --exclude ".git" -r -e close_write . ; kill $$PID ; done)

db-reset:
	psql -h localhost -U sfe -c "drop schema public cascade"
	psql -h localhost -U sfe -c "create schema public"

test:
	go test $(ARGS) ./...

docker-deps:
	$(MAKE) -C postgres-docker docker-build
	$(MAKE) -C postgres-docker run-prod

docker-build:
	docker build -t sfe .

docker-run:
	docker start sfe-db # Did you run 'make docker-deps'?
	-docker top sfe-app && docker rm -f sfe-app
	docker run -d -p 8111:8000 --name sfe-app --link sfe-db:sfe-db sfe # Did you run 'make docker-build?'

docker: docker-build docker-run

# Must specify TO.
deploy-deps: check-to
	rsync -azP . $(TO):~/syncfbevents
	ssh $(TO) 'cd ~/syncfbevents && make docker-deps'

# Must specify TO.
deploy: check-to
	rsync -azP . $(TO):~/syncfbevents
	ssh $(TO) 'cd ~/syncfbevents && make docker'

check-to:
	ifndef TO
	    $(error TO is undefined)
	endif
