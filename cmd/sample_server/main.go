package main

import (
	"github.com/jonhanks/goweb"
	"log"
	"net/http"
)

func indexHandler(writer http.ResponseWriter, request* http.Request) {
	writer.Write([]byte("hello world!"))
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", goweb.Chain(indexHandler, goweb.NewLoggingMiddleware))
	log.Fatal(http.ListenAndServe(":8080", mux))
}
