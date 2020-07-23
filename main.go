package main

import "flag"
import "net/http"
import "os"
import "fmt"
import "time"

func die(v ...interface{}) {
	fmt.Println(v...)
	os.Exit(1)
}

func main() {
	addr := flag.String("addr", "localhost:8080", "address to serve on")
	freq := flag.String("gc-freq", "5m", "gc frequency")

	flag.Parse()

	duration, err := time.ParseDuration(*freq)
	if err != nil {
		die("invalid duration:", err)
	}

	if duration <= 0 {
		die("invalid duration (<=0)")
	}

	s := NewServer(duration)
	go s.GCLoop()
	http.ListenAndServe(*addr, s.mux)
}
