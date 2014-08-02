This package implements a library to receive Server Sent Events.  
Note that the package name is "sse".  

For details see: http://en.wikipedia.org/wiki/Server-sent_events  

[![GoDoc](https://godoc.org/github.com/billhathaway/serverSentEvents?status.png)](https://godoc.org/github.com/billhathaway/serverSentEvents)  

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

