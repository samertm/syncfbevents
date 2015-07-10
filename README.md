# Sync FB Events

Let's get this to work.

Next:
 - [x] Barf out the events somewhere.
 - [ ] Create a webcal link for the events.
 - [ ] Sync them with Google Calendar.

# Notes

// do I need time.ParseInLocation?
// StartTime:2015-07-21T19:30:00-0700
// StartTime:2025-12-29

// SAMER: Remove timezone.
// SAMER: Make unique on id. Or... do I have an off-by-one bug?

Deploy on heroku?

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

// func main() {
// 	var woTimeLayout = "2006-01-02"
// 	var wTimeLayout = "2006-01-02T15:04:05-0700"
// 	var woTime = "2025-12-29"
// 	var wTime = "2015-07-21T19:30:00-0700" // zoopolis
// 	fmt.Println(time.Parse(woTimeLayout, woTime))
	
// 	t, _ := time.Parse(wTimeLayout, wTime)
// 	fmt.Println(t.UTC().Format("20060102T150405Z"))
// }


# Setup

$ pacman -S postgres
$ sudo -i -u postgres initdb --locale en_US.UTF-8 -E UTF8 -D '/var/lib/postgres/data'
$ sudo systemctl start postgresql
$ sudo systemctl enable postgresql
$ sudo -i -u postgres createuser --interactive <your-system-username>
$ createdb syncfbevents

# LICENSE

The license for this project is AGPLv3.
