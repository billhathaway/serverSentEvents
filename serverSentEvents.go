// Package sse handles Server Sent Events  
// See http://www.w3.org/TR/eventsource/#the-eventsource-interface  

// TODO: handle various retry logic better  
package sse

import (
	"bufio"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Event maps to a single SSE
type Event struct {
	Type      string
	Data      string
	LastID    string
	populated bool
}

const (
	// EventStream is the only permissible content type to handle
	EventStream = "text/event-stream"
)

var (
	// fieldRegexp matches pattern lines containing fields must conform to
	fieldRegexp = regexp.MustCompile("^([a-z]+): (.+)$")

	// MaxBadLines indicates the number of illegal lines returned that cause the connection to error out
	MaxBadLines = 10
	// MaxRetries is the number of bad HTTP responses in a row we can get before we give up
	MaxRetries = 3

	minReconnectionInterval = time.Duration(100 * time.Millisecond)
	maxReconnectionInterval = time.Minute
)

type (
	// Listener provides the event stream
	Listener struct {
		client               *http.Client
		request              *http.Request
		badLines             int
		lastID               string
		reconnectionInterval time.Duration
		C                    chan Event
		retries              int
	}
)

func (l *Listener) getStream() (*http.Response, error) {
	for {
		response, err := l.client.Do(l.request)
		// TODO: should be checking for error types and retrying certain ones
		if err != nil {
			return nil, err
		}
		switch response.StatusCode {
		case http.StatusOK:
			if response.Header.Get("Content-Type") == EventStream {
				l.reconnectionInterval = minReconnectionInterval
				l.retries = 0
				return response, nil
			}
			return nil, fmt.Errorf("Invalid Content-Type")

		case http.StatusInternalServerError, http.StatusBadGateway, http.StatusGatewayTimeout:
			response.Body.Close()
			l.retries++
			if l.retries > MaxRetries {
				fmt.Println("Max retries exceeded")
				return nil, fmt.Errorf("Max retries exceeded")
			}
			fmt.Println("Bad response, retrying")
			break
		}
		time.Sleep(l.reconnectionInterval)
		l.reconnectionInterval *= 2
		if l.reconnectionInterval > maxReconnectionInterval {
			l.reconnectionInterval = maxReconnectionInterval
		}
	}
}

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

// TODO: handle error cases better
func (l *Listener) readEvents(response *http.Response) {
	defer close(l.C)
	for {
		badLineCount := 0
		scanner := bufio.NewScanner(response.Body)
		event := Event{}
		for scanner.Scan() {
			line := scanner.Text()
			switch {
			case strings.HasPrefix(line, ":"):
				// ignore lines starting with colon per spec
			case strings.HasPrefix(line, "id: "):
				event.LastID = parseLine(line)
				l.lastID = event.LastID
			case strings.HasPrefix(line, "event: "):
				event.Type = parseLine(line)
				event.populated = true
			case strings.HasPrefix(line, "retry: "):
				retryTimeString := parseLine(line)
				retryTimeMS, err := strconv.Atoi(retryTimeString)
				if err != nil {
					l.reconnectionInterval = time.Millisecond * time.Duration(retryTimeMS)
				}
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
					l.C <- event
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
		response.Body.Close()
		var err error
		response, err = l.getStream()
		if err != nil {
			return
		}
	}
}

// ListenReq sends the http.Request,  starts listening for Server Sent Events and sends them to the returned channel
func ListenReq(req *http.Request) (*Listener, error) {
	l := &Listener{}
	l.client = &http.Client{}
	l.request = req
	l.reconnectionInterval = minReconnectionInterval
	l.C = make(chan Event)
	response, err := l.getStream()
	if err != nil {
		return nil, err
	}
	go func() {
		l.readEvents(response)
	}()
	return l, nil
}

// Listen sends a GET request based on the URL, starts listening for Server Sent Events and sends them to the returned channel
func Listen(url string) (*Listener, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", EventStream)
	return ListenReq(request)
}
