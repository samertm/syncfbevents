package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/context"
	"github.com/huandu/facebook"
	"github.com/samertm/syncfbevents/conf"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
	"golang.org/x/oauth2"
	facebookoauth "golang.org/x/oauth2/facebook"
)

var errorTemplate = initializeTemplate("templates/error.html")

type errorTemplateVars struct {
	Code    int
	Message string
}

func initializeTemplate(file string) *template.Template {
	return template.Must(template.ParseFiles("templates/layout.html", file))
}

var indexTemplate = initializeTemplate("templates/index.html")

type indexTemplateVars struct {
	Name        string
	CalendarURL string
}

func serveIndex(c web.C, w http.ResponseWriter, r *http.Request) error {
	s := getSession(c)
	u := getUser(s) // SAMER: Change to auth.
	v := indexTemplateVars{}
	if u != nil {
		v.Name = u.Name
		// SAMER: Reverse router?
		rawurl := absoluteURL(fmt.Sprintf("/calendar/%s", u.FacebookID))
		calURL, err := url.Parse(rawurl)
		if err != nil {
			return err
		}
		calURL.Scheme = "webcal"
		v.CalendarURL = calURL.String()
	}
	return indexTemplate.Execute(w, v)
}

func serveLogin(c web.C, w http.ResponseWriter, r *http.Request) error {
	url := oauthConf.AuthCodeURL(oauthStateString, oauth2.AccessTypeOnline)
	return HTTPRedirect{
		To:   url,
		Code: http.StatusTemporaryRedirect,
	}
}

func serveFacebookCallback(c web.C, w http.ResponseWriter, r *http.Request) error {
	s := getSession(c)
	state := r.FormValue("state")
	if state != oauthStateString {
		return fmt.Errorf("invalid oauth state, expected '%s', got '%s'\n", oauthStateString, state)
	}

	code := r.FormValue("code")
	token, err := oauthConf.Exchange(oauth2.NoContext, code)
	if err != nil {
		return fmt.Errorf("oauthConf.Exchange() failed with '%s'\n", err)
	}

	// Save token here.
	res, err := facebook.Get("/me", facebook.Params{
		"access_token": token.AccessToken,
	})
	if err != nil {
		return fmt.Errorf("fb.Get() failed with '%s'\n", err)
	}
	log.Printf("Logged in with Facebook user: %s\n", res["name"])
	var v struct {
		Name string `facebook:",required"`
		ID   string `facebook:",required"`
	}
	if err := res.Decode(&v); err != nil {
		return err
	}
	// Save user to DB.
	u, err := GetCreateUser(v.Name, v.ID)
	if err != nil {
		return fmt.Errorf("Could not create user %s: %s", res["name"], err)
	}
	// SAMER: Replace with fb.ExchangeToken.
	// Get the longer lasting token.
	var longLivedTokenURL = facebookoauth.Endpoint.TokenURL + "?" +
		"grant_type=fb_exchange_token&" +
		"client_id=" + oauthConf.ClientID + "&" +
		"client_secret=" + oauthConf.ClientSecret + "&" +
		"fb_exchange_token=" + token.AccessToken
	resp, err := http.Get(longLivedTokenURL)
	if err != nil {
		return fmt.Errorf("Could not extend access token for %s: %s", v.Name, err)
	}
	b, _ := ioutil.ReadAll(resp.Body)
	m, err := url.ParseQuery(string(b))
	if err != nil {
		return fmt.Errorf("Could not parse long-lived token for %s: %s", v.Name, err)
	}
	if len(m["access_token"]) == 0 || len(m["expires"]) == 0 {
		return fmt.Errorf("Values missing from long-lived token response for %s: %s", v.Name, err)
	}
	if err := SetAccessToken(u, m["access_token"][0], m["expires"][0]); err != nil {
		return fmt.Errorf("Could not set long-lived access token for %s: %s", v.Name, err)
	}
	s.Values[userIDSessionKey] = u.ID
	if err := s.Save(r, w); err != nil {
		log.Println(err)
	}
	return HTTPRedirect{To: "/", Code: http.StatusTemporaryRedirect}
}

// Handles "raw" query by spitting out raw Events list.
func serveCalendar(c web.C, w http.ResponseWriter, r *http.Request) error {
	fbID := c.URLParams["fbID"]
	u, err := GetUser(UserSpec{FacebookID: fbID})
	if err != nil {
		return err
	}
	// Check to see if the user has a valid access token.
	if !u.AccessToken.Valid || u.ExpiresOn == nil || u.ExpiresOn.Before(time.Now()) {
		return fmt.Errorf("The access token for %d is not valid", u.ID)
	}
	fbSession := fb.Session(u.AccessToken.String)
	res, err := fbSession.Get("/me/events?"+
		"fields=owner,end_time,description,id,name,rsvp_status,"+
		"place,start_time,timezone", nil)
	if err != nil {
		return fmt.Errorf("Could not get events: %s", err)
	}
	pr, err := res.Paging(fbSession)
	if err != nil {
		return fmt.Errorf("Could not use as paging result: %s", err)
	}
	var allEvents []Event
	dateMarker := time.Now().Add(-48 * time.Hour)
outer:
	for {
		for _, r := range pr.Data() {
			var e Event
			if err := r.Decode(&e); err != nil {
				return fmt.Errorf("Could not decode events: %s", err)
			}
			dt, err := parseFacebookDateTime(e.StartTime)
			if err != nil {
				return fmt.Errorf("Error during calendar generation: %s", err)
			}
			e.startTimeParsed = dt
			if e.startTimeParsed.T.Before(dateMarker) {
				break outer
			}
			if e.RSVPStatus == "attending" {
				allEvents = append(allEvents, e)
			}
		}
		if !pr.HasNext() {
			break
		}
		pr.Next()
	}
	if len(r.URL.Query()["raw"]) > 0 {
		// The URL has the query "raw".
		w.Write([]byte(fmt.Sprintf("%+v", allEvents)))
		return nil
	}
	cal, err := generateICal(fbID, u.Name, allEvents)
	if err != nil {
		return fmt.Errorf("Error generating calendar for %s: %s", fbID, err)
	}
	w.Write(cal)
	return nil
}

var privacyPolicyTemplate = initializeTemplate("templates/privacypolicy.html")

func servePrivacyPolicy(c web.C, w http.ResponseWriter, r *http.Request) error {
	return privacyPolicyTemplate.Execute(w, nil)
}

// SAMER: Convert to CLRF at some point?
func generateICal(fbID, userName string, es []Event) ([]byte, error) {
	buf := &bytes.Buffer{}
	buf.WriteString(`BEGIN:VCALENDAR
PRODID:-//Sync Events//NONSGML Sync Events V1.0//EN
X-WR-CALNAME:` + userName + `'s Facebook Events -- Attending Only
VERSION:2.0
CALSCALE:GREGORIAN
METHOD:PUBLISH
`)
	for _, e := range es {
		// Remember: every line must end with \n.
		buf.WriteString("BEGIN:VEVENT\n")
		var owner string
		if e.Owner.Name == "" {
			owner = "No Owner"
		} else {
			owner = e.Owner.Name
		}
		buf.WriteString(fmt.Sprintf("ORGANIZER:CN=%s:MAILTO:noreply@facebookmail.com\n", owner))
		iCalStart, iCalEnd, err := toICalDateTime(e.startTimeParsed, e.EndTime)
		if err != nil {
			return nil, err
		}
		buf.WriteString(fmt.Sprintf("DTSTART:%s\n", iCalStart))
		buf.WriteString(fmt.Sprintf("DTEND:%s\n", iCalEnd))
		// UID: Hash the facebook ID with the user's ID.
		buf.WriteString(fmt.Sprintf("UID:%s\n", generateICalUID(fbID, e.ID)))
		buf.WriteString(fmt.Sprintf("SUMMARY:%s\n", toICalText(e.Name)))
		if e.Place.Name != "" {
			buf.WriteString(fmt.Sprintf("LOCATION:%s\n", toICalText(e.Place.Name)))
		}
		buf.WriteString(fmt.Sprintf("URL:%s\n", e.FacebookURL()))
		if e.Description != "" {
			buf.WriteString("DESCRIPTION:")
			var d = e.Description + "\n\n" + e.FacebookURL()
			buf.Write(toICalTextLimited(d, 34))
			buf.WriteRune('\n')
		}
		buf.WriteString("CLASS:PUBLIC\n")
		buf.WriteString("STATUS:CONFIRMED\n")
		buf.WriteString("PARTSTAT:ACCEPTED\n")
		buf.WriteString("END:VEVENT\n")
	}
	buf.WriteString("END:VCALENDAR\n")
	return buf.Bytes(), nil
}

func generateICalUID(fbID, eventID string) string {
	hasher := md5.New()
	hasher.Write([]byte(fbID))
	hasher.Write([]byte(eventID))
	return hex.EncodeToString(hasher.Sum(nil)) + "@syncfbevents.com"
}

var FacebookDateTimeLayout = "2006-01-02T15:04:05-0700"
var FacebookDateTimeLayoutAlt = "2006-01-02T15:04:05" // Used pre-2013.
var FacebookDateLayout = "2006-01-02"

var ICalDateTimeLayout = "20060102T150405Z"
var ICalDateLayout = "20060102"

type DateTime struct {
	T        time.Time
	DateOnly bool
}

func parseFacebookDateTime(datetime string) (DateTime, error) {
	// First, we try to parse the startTime with a date and time.
	dt, err := time.Parse(FacebookDateTimeLayout, datetime)
	if err != nil {
		// The date may be in the alternate format.
		dtAlt, err := time.Parse(FacebookDateTimeLayoutAlt, datetime)
		if err != nil {
			// The date may be in the date layout.
			d, err := time.Parse(FacebookDateLayout, datetime)
			if err != nil {
				return DateTime{}, fmt.Errorf("Could not parse datetime %s: %s", datetime, err)
			}
			return DateTime{T: d, DateOnly: true}, nil
		}
		return DateTime{T: dtAlt, DateOnly: false}, nil
	}
	return DateTime{T: dt, DateOnly: false}, nil
}

// startTime must be 2006-01-02 or 2006-01-02T15:04:05-0700. endTime
// must be in a similar format or empty. Returns a time string in a
// valid ICal UTC Date-Time format.
//
// If endTime is empty, then iCalEnd is one day after iCalStart if
// iCalStart does not include a time, otherwise it is set to three
// hours after iCalStart.
func toICalDateTime(startDateTime DateTime, endTime string) (iCalStart, iCalEnd string, err error) {
	if startDateTime.DateOnly {
		d := startDateTime.T
		// d is valid, format it as just a date.
		iCalStart = d.UTC().Format(ICalDateLayout)
		if endTime == "" {
			// iCalEnd needs to be a day after iCalStart.
			iCalEnd = d.UTC().Add(24 * time.Hour).Format(ICalDateLayout)
			return iCalStart, iCalEnd, nil
		}
		// endTime is nonempty.
		endD, err := parseFacebookDateTime(endTime)
		if err != nil {
			return "", "", fmt.Errorf("Error parsing endTime (%s): %s", endTime, err)
		}
		// iCalEnd must be the same value type as iCalStart,
		// i.e. only a date.
		iCalEnd = endD.T.UTC().Format(ICalDateLayout)
		return iCalStart, iCalEnd, nil
	}
	dt := startDateTime.T
	// dt is valid, format it as a date and time.
	iCalStart = dt.UTC().Format(ICalDateTimeLayout)
	if endTime == "" {
		// iCalEnd needs to be three hours after iCalStart.
		iCalEnd = dt.UTC().Add(3 * time.Hour).Format(ICalDateTimeLayout)
		return iCalStart, iCalEnd, nil
	}
	// endTime is nonempty, and must have the same value type as
	// startTime, i.e. a date-time.
	endDT, err := parseFacebookDateTime(endTime)
	if err != nil {
		return "", "", fmt.Errorf("Error parsing endTime (%s): %s", endTime, err)
	}
	iCalEnd = endDT.T.UTC().Format(ICalDateTimeLayout)
	return iCalStart, iCalEnd, nil
}

// Convert text to iCal-prepared text.
//       text       = *(TSAFE-CHAR / ":" / DQUOTE / ESCAPED-CHAR)
//          ; Folded according to description above
//
//       ESCAPED-CHAR = ("\\" / "\;" / "\," / "\N" / "\n")
//          ; \\ encodes \, \N or \n encodes newline
//          ; \; encodes ;, \, encodes ,
//
//       TSAFE-CHAR = WSP / %x21 / %x23-2B / %x2D-39 / %x3C-5B /
//                    %x5D-7E / NON-US-ASCII
//          ; Any character except CONTROLs not needed by the current
//          ; character set, DQUOTE, ";", ":", "\", ","
func toICalText(text string) []byte {
	return toICalTextLimited(text, 0)
}

// toICalTextLimited returns an iCal-prepared text. If lineLength is 0
// or less, the lines are not limited. Otherwise, lines are limited to
// around lineLength.
func toICalTextLimited(text string, lineLength int) []byte {
	var lines [][]byte
	buf := &bytes.Buffer{}
	for _, rune := range text {
		// Check for escaped characters.
		switch rune {
		case '\\':
			buf.WriteString("\\\\")
		case ';':
			buf.WriteString("\\;")
		case ',':
			buf.WriteString("\\,")
		case '\n':
			buf.WriteString("\\n")
		default:
			buf.WriteRune(rune)
		}
		if lineLength > 0 && buf.Len() >= lineLength {
			// Flush buf and wipe.
			lines = append(lines, buf.Bytes())
			buf = &bytes.Buffer{}
		}
	}
	if buf.Len() != 0 { // SAMER: I think I need this.
		lines = append(lines, buf.Bytes())
	}
	return bytes.Join(lines, []byte("\n "))
}

type Event struct {
	ID          string     `facebook:"id,required"`
	Name        string     `facebook:"name,required"`
	StartTime   string     `facebook:"start_time,required"`
	EndTime     string     `facebook:"end_time"`
	RSVPStatus  string     `facebook:"rsvp_status"`
	Description string     `facebook:"description"`
	Owner       EventOwner `facebook:"owner"`
	Place       EventPlace `facebook:"place"`

	startTimeParsed DateTime // Does not exist in Facebook's output.
}

func (e Event) FacebookURL() string {
	return fmt.Sprintf("https://www.facebook.com/events/%s/", e.ID)
}

type EventOwner struct {
	Name string `facebook:"name"`
}

type EventPlace struct {
	Name string `facebook:"name"`
}

var (
	oauthConf = &oauth2.Config{
		ClientID:     conf.Config.FacebookID,
		ClientSecret: conf.Config.FacebookSecret,
		RedirectURL:  conf.Config.BaseURL + "/facebook_callback",
		Scopes:       []string{"user_events"},
		Endpoint:     facebookoauth.Endpoint,
	}
	oauthStateString = conf.Config.OAuthStateString
	fb               = facebook.New(conf.Config.FacebookID, conf.Config.FacebookSecret)
)

func main() {
	// Serve static files.
	staticDirs := []string{"bower_components", "res"}
	for _, d := range staticDirs {
		static := web.New()
		pattern, prefix := fmt.Sprintf("/%s/*", d), fmt.Sprintf("/%s/", d)
		static.Get(pattern, http.StripPrefix(prefix, http.FileServer(http.Dir(d))))
		http.Handle(prefix, static)
	}

	goji.Use(applySessions)
	goji.Use(context.ClearHandler)

	goji.Get("/", handler(serveIndex))
	goji.Get("/login", handler(serveLogin))
	goji.Get("/facebook_callback", handler(serveFacebookCallback))
	goji.Get("/calendar/:fbID", handler(serveCalendar))
	goji.Get("/privacypolicy", handler(servePrivacyPolicy))
	goji.Get("/privacypolicy.html", handler(servePrivacyPolicy))
	goji.Serve()
}
