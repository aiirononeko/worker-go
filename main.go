package main

import (
	"net/http"

	"github.com/syumai/workers"
)

func main() {
	http.HandleFunc("/hello", hello)
	workers.Serve(nil) // use http.DefaultServeMux
}

func hello(w http.ResponseWriter, req *http.Request) {
	msg := "Hello, BulkTrack!"
	w.Write([]byte(msg))
}
