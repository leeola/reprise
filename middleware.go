package reprise

import (
	"log"
	"net/http"
)

func Middleware(rep *Reprise) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			if _, err := rep.WriteRequest(r); err != nil {
				log.Printf("reprise middleware writerequest: %v", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			// serve the request
			next.ServeHTTP(w, r)

		}

		return http.HandlerFunc(fn)
	}
}
