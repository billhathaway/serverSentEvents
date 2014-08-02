This package implements a library to receive Server Sent Events.  

For details see: http://en.wikipedia.org/wiki/Server-sent_events  

    package main

    import "fmt"
    import "github.com/billhathaway/serverSentEvents"

    func main() {
    	listener, err := sse.Listen("http://somewhere/")
    	if err != nil {
    		fmt.Printf("Problem getting events - %s\n", err.Error())
    		return
    	}
    	for event := range listener.C {
    		fmt.Printf("Received event %s\n", event.String())
    	}
    }

