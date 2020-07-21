package main

import "log"
import "flag"
import "net/http"
import "time"

func main() {
	addr := flag.String("addr", "localhost:8080", "address to serve on")
	freq := flag.Int64("gc-freq", 5*60, "gc frequency (in seconds)")

	flag.Parse()

	if *freq <= 0 {
		log.Fatal("invalid frequency: ", *freq)
	}

	s := NewServer(time.Duration(*freq) * time.Second)
	go s.Loop()
	http.ListenAndServe(*addr, s.mux)
}
