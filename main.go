package main

import (
	"flag"
	"github.com/eugene-eeo/emq/tctx2"
	"github.com/satori/go.uuid"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", ":8080", "TCP listening address")
	flag.Parse()

	mux := http.NewServeMux()
	srv := &server{
		tasks:      map[uuid.UUID]*QueueNode{},
		queues:     map[string]*Queue{},
		router:     mux,
		waiters:    &Waiters{},
		dispatched: make(chan TaskInfo),
		context:    tctx2.NewContext(),
		version:    Version{Version: "0.1.0-alpha"},
	}
	srv.routes()
	go srv.listenDispatched()
	log.Fatal(http.ListenAndServe(*addr, mux))
}
