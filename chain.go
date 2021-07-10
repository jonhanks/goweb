package goweb

import "net/http"

type Generator func(handlerFunc http.HandlerFunc) http.HandlerFunc

func Chain(handler http.HandlerFunc, generators ...Generator) http.HandlerFunc {
	for i := len(generators)-1; i >= 0; i-- {
		handler = generators[i](handler)
	}
	return handler
}
