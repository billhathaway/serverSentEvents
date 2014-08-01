package sse_test

import (
	"fmt"
	"github.com/billhathaway/serverSentEvents"
	"net/http"
	"net/http/httptest"
	"testing"
)

func init() {
	server = httptest.NewServer(sseGenerator{})
}

var (
	server *httptest.Server
)

type sseGenerator struct{}

// ServeHTTP sends back 3 events for testing
func (sseGenerator) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "event: add\ndata: 73857293\n\n")
	fmt.Fprintf(w, "event: remove\n: skip comment\ndata: 2153\n\n")
	fmt.Fprintf(w, "event: doubleLine\ndata: line1\ndata: line2\n\n")
}

// Verify that 3 events are ready back properly.  The first event is standard, the second event contains a comment in the middle, and the third event has two data lines
func TestSSE(t *testing.T) {
	localServer := fmt.Sprintf("http://%s/", server.Listener.Addr())
	eventChan, err := sse.Listen(localServer)
	if err != nil {
		t.Log(err.Error)
		t.FailNow()
	}
	events := []sse.Event{sse.Event{Type: "add", Data: "73857293"}, sse.Event{Type: "remove", Data: "2153"}, sse.Event{Type: "doubleLine", Data: "line1\rline2"}}
	index := 0
	for event := range eventChan {
		if event.Type == events[index].Type && event.Data == events[index].Data {
			t.Logf("Found expected event %s\n", event.String())
		} else {
			t.Logf("Did not match expected event found=[%s] expected=\n[%s]\n", event.String(), events[index].String())
			t.Fail()
		}
		index++
	}
	server.Close()
}
