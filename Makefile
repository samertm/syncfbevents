.PHONY: serve test db-reset deps watch-serve

serve:
	go install github.com/samertm/syncfbevents
	syncfbevents

watch-serve:
	$(shell while true; do $(MAKE) serve & PID=$$! ; echo $$PID ; inotifywait --exclude ".git" -r -e close_write . ; kill $$PID ; done)

deps:
	go get github.com/codegangsta/gin
	echo "Make sure you set up Postgres correctly."

db-reset:
	psql -d syncfbevents -c "drop schema public cascade"
	psql -d syncfbevents -c "create schema public"

test:
	go test ./...
