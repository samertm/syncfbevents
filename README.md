# Sync FB Events

# Notes on iCal format.

```
BEGIN:VEVENT
ORGANIZER;CN=<Owner>:MAILTO:noreply@facebookmail.com
DTSTART - either yyyymmdd or yyyymmddThhmmssZ converted to UTC
DTEND - same as DTSTART if it doesn't exist.
UID - must be there, must be unique, should add '@domain.com' to the end.
SUMMARY - name
LOCATION - location name
URL - https://www.facebook.com/events/:event_id/
DESCRIPTION - description (how is it formatted?)
CLASS:PUBLIC
STATUS:CONFIRMED
PARTSTAT:ACCEPTED
END:VEVENT
```

# Setup

```bash
$ sudo pacman -S postgres # Set up database.
$ sudo -i -u postgres initdb --locale en_US.UTF-8 -E UTF8 -D '/var/lib/postgres/data'
$ sudo systemctl start postgresql
$ sudo systemctl enable postgresql
$ sudo -i -u postgres createuser --interactive <your-system-username>
$ createdb syncfbevents
$ sudo pacman -S docker # Install docker.
$ sudo gpasswd -a <USER> docker && echo "Login and then logout."
```

# Run

### Serve

```bash
$ make server      # Serve site.
$ make watch-serve # Serve site, restarting when any files in the
                   # project directory change (requires inotifywait).
```

### Auxiliary

```bash
$ make db-reset # Wipe the database.
$ make test     # Run tests.
```

### Docker

```bash
$ make docker-deps  # Pull, configure, and run the docker container dependencies (only run once).
$ make docker-build # Build the docker container for the app.
$ make docker-run   # Run the docker container as a daemon.
```

### Deploy

```bash
$ make deploy-deps TO=<SSH-NAME> # Pull docker deps on SSH-NAME (only run once).
$ make deploy TO=<SSH-NAME>      # Push code to server and run docker.
```

# LICENSE

The license for this project is AGPLv3.
