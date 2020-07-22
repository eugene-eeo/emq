package main

import "log"
import "flag"
import "net/http"
import "time"

func main() {
	addr := flag.String("addr", "localhost:8080", "address to serve on")
	freq := flag.String("gc-freq", "5m", "gc frequency")

	flag.Parse()

	duration, err := time.ParseDuration(*freq)
	if err != nil {
		log.Fatal("invalid duration:", err)
	}

	if duration <= 0 {
		log.Fatal("invalid duration (<=0)")
	}

	s := NewServer(duration)
	go s.Loop()
	http.ListenAndServe(*addr, s.mux)
}
