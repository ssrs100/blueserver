package middleware

import (
	"github.com/dimfeld/httptreemux"
	"net/http"
)

func PassThrough(fn httptreemux.HandlerFunc) httptreemux.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, ps map[string]string) {
		fn(w, r, ps)
	}
}
