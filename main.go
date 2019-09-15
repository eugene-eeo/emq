package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	fwd := make(chan TaskInfo)
	srv := &server{
		tasks:   map[string]*Task{},
		queues:  map[string]*Queue{},
		router:  mux,
		waiters: Waiters{},
		version: Version{
			Version: "0.1.0-alpha",
		},
		dispatched: NewDispatched(fwd),
	}
	srv.routes()
	go srv.listenDispatched()
	log.Fatal(http.ListenAndServe(":8080", mux))
}
