package main

import "net/http"

func (app *application) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := app.authenticateToken(r)
		if err != nil {
			app.InvalidCredentials(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}
