package reprise

import (
	"log"
	"net/http"
)

func Middleware(repFunc RepriseFunc) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			rep, ok := repFunc(r)
			if !ok {
				// serve the request with no modification
				next.ServeHTTP(w, r)
				return
			}

			repReq, err := NewRequest(r)
			if err != nil {
				log.Printf("reprise middleware newrequest: %v", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			tee, err := NewResponseWriterTee(w, r)
			if err != nil {
				log.Printf("reprise middleware newresponsetee: %v", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			// serve the request
			next.ServeHTTP(tee, r)

			reqRes, err := tee.Response()
			if err != nil {
				log.Printf("reprise middleware response: %v", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			if err := rep.Write(reqRes, repReq); err != nil {
				log.Printf("reprise middleware write: %v", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
		}

		return http.HandlerFunc(fn)
	}
}

type RepriseFunc func(*http.Request) (*Reprise, bool)

func All(r *Reprise) RepriseFunc {
	return RepriseFunc(func(*http.Request) (*Reprise, bool) {
		return r, true
	})
}
