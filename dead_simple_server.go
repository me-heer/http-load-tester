package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func blazinglyFastHandler(w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprintf(w, "%s, you are blazingly fast.", r.RemoteAddr)
	time.Sleep(time.Millisecond * 300) // each request takes 300 ms
	if err != nil {
		log.Fatal("Unfortunately, I wasn't blazingly fast")
	}
}

func main() {
	http.HandleFunc("/", blazinglyFastHandler)
	err := http.ListenAndServe("127.0.0.1:12345", nil)
	if err != nil {
		println(err.Error())
		log.Fatal("I couldn't serve")
	}
}
