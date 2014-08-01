// Package sse handles Server Sent Events
// See http://www.w3.org/TR/eventsource/#the-eventsource-interface
// TODO: handle reconnects
package sse

import (
	"bufio"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// Event maps to a single SSE
type Event struct {
	Type      string
	Data      string
	LastID    string
	populated bool
}

var (
	fieldRegexp = regexp.MustCompile("^([a-z]+): (.+)$")
	// MaxBadLines indicates the number of illegal lines we see that cause the connection to error out
	MaxBadLines = 100
)

// String generates a string representation of an Event
func (e Event) String() string {
	if e.LastID != "" {
		return fmt.Sprintf("type=%s data=%s id=%s", e.Type, e.Data, e.LastID)
	}
	return fmt.Sprintf("type=%s data=%s", e.Type, e.Data)
}

// parseLine returns the data portion of a line (after the $field:)
func parseLine(data string) string {
	if fieldRegexp.MatchString(data) {
		return fieldRegexp.FindStringSubmatch(data)[2]
	}
	return ""
}

// TODO: once we start handling auto-reconnect do not close the eventChan
// when the current *http.Response gets an error
func readEvents(response *http.Response, eventChan chan Event) {
	defer response.Body.Close()
	defer close(eventChan)
	var badLineCount int
	scanner := bufio.NewScanner(response.Body)
	event := Event{}

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, ":"):
			// ignore lines starting with colon per spec
		case strings.HasPrefix(line, "id: "):
			event.LastID = parseLine(line)
		case strings.HasPrefix(line, "event: "):
			event.Type = parseLine(line)
			event.populated = true
		case strings.HasPrefix(line, "data: "):
			if event.Data == "" {
				event.Data = parseLine(line)
			} else {
				// if there are multiple data lines, always append a LF to existing data and then append the new data
				event.Data += "\r" + parseLine(line)
			}
			event.populated = true
		case len(line) == 0:
			if event.populated {
				eventChan <- event
				event = Event{LastID: event.LastID}
				event.populated = false
			}
		default:
			badLineCount++
			if badLineCount > MaxBadLines {
				return
			}
		}
	}

}

// ListenReq sends the http.Request and starts listening for Server Sent Events and sends them to the returned channel
func ListenReq(req *http.Request) (chan Event, error) {
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	eventChan := make(chan Event)
	go readEvents(response, eventChan)
	return eventChan, nil
}

// Listen sends a GET request based on the URL and starts listening for Server Sent Events and sends them to the returned channel
func Listen(url string) (chan Event, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	eventChan := make(chan Event)
	go readEvents(response, eventChan)
	return eventChan, nil
}
