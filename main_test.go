package main

import "testing"

func TestToICalDateTime(t *testing.T) {
	// Three test cases:
	//  - only startTime date
	//  - only startTime date-time
	//  - startTime date-time and endTime date-time
	// All correct values are from Facebook's iCal.
	vs := []struct {
		startTime     string
		endTime       string
		wantICalStart string
		wantICalEnd   string
	}{
		{
			startTime:     "2025-12-29",
			wantICalStart: "20251229",
			wantICalEnd:   "20251230",
		},
		{
			startTime:     "2015-07-23T19:00:00-0700",
			wantICalStart: "20150724T020000Z",
			wantICalEnd:   "20150724T050000Z",
		},
		{
			startTime:     "2015-07-12T08:30:00-0700",
			endTime:       "2015-07-12T15:00:00-0700",
			wantICalStart: "20150712T153000Z",
			wantICalEnd:   "20150712T220000Z",
		},
	}
	//toICalDateTime
	for _, v := range vs {
		iCalStart, iCalEnd, err := toICalDateTime(v.startTime, v.endTime)
		if err != nil {
			t.Error(err)
			continue
		}
		if iCalStart != v.wantICalStart {
			t.Errorf("Wanted iCalStart %s, got %s", v.wantICalStart, iCalStart)
		}
		if iCalEnd != v.wantICalEnd {
			t.Errorf("Wanted iCalEnd %s, got %s", v.wantICalEnd, iCalEnd)
		}
	}
}
