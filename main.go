package main

import (
	"github.com/eugene-eeo/emq/tctx2"
	"github.com/satori/go.uuid"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	srv := &server{
		tasksById:  map[uuid.UUID]*Task{},
		queues:     map[string]*Queue{},
		router:     mux,
		waiters:    &Waiters{},
		dispatched: make(chan TaskInfo),
		context:    tctx2.NewContext(),
		version:    Version{Version: "0.1.0-alpha"},
	}
	srv.routes()
	go srv.listenDispatched()
	log.Fatal(http.ListenAndServe(":8080", mux))
}
