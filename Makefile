.PHONY: serve test db-reset

serve:
	go install github.com/samertm/syncfbevents
	syncfbevents

db-reset:
	psql -d syncfbevents -c "drop schema public cascade"
	psql -d syncfbevents -c "create schema public"

test:
	go test ./...
