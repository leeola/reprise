package reprise

import (
	"log"
	"net/http"
)

func Middleware(rep *Reprise) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			repReq, err := NewRequest(r)
			if err != nil {
				log.Printf("reprise middleware newrequest: %v", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			// serve the request
			next.ServeHTTP(w, r)

			reqRes := Response{}

			if err := rep.Write(reqRes, repReq); err != nil {
				log.Printf("reprise middleware write: %v", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
		}

		return http.HandlerFunc(fn)
	}
}
