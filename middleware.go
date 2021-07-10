package goweb

import (
	"log"
	"net/http"
)

func Logging() bool {
	return true
}


type statusCodeWriter struct {
	w http.ResponseWriter
	statusCode int
}

func newStatusCodeWriter(w http.ResponseWriter) *statusCodeWriter {
	return &statusCodeWriter{w:w}
}

func (l*statusCodeWriter) setStatus(status int) {
	if l.statusCode == 0 {
		l.statusCode = status
	}
}

func (l*statusCodeWriter) Header() http.Header {
	return l.w.Header()
}

func (l*statusCodeWriter) Write(bytes []byte) (int, error) {
	l.setStatus(http.StatusOK)
	return l.w.Write(bytes)
}

func (l*statusCodeWriter) WriteHeader(statusCode int) {
	l.setStatus(statusCode)
	l.w.WriteHeader(statusCode)
}

func (l* statusCodeWriter) getStatusCode() int {
	l.setStatus(http.StatusOK)
	return l.statusCode
}

func LoggingMiddleware(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		sw := newStatusCodeWriter(writer)
		handlerFunc(sw, request)
		log.Printf("%s - %d", request.URL.Path, sw.statusCode)
	}
}

func NewLoggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r* http.Request) {
		sw := newStatusCodeWriter(w)
		next(sw, r)
		log.Printf("%s - %d", r.URL.Path, sw.statusCode)
	}
}

func NewPanicHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r* http.Request) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovering from a panic, %v", r)
			}
		}()
		next(w, r)
	}
}