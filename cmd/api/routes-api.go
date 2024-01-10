package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func (app *application) routes() http.Handler {
	mux := chi.NewRouter()

	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://*", "https://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	mux.Post("/api/payment-intent", app.GetPaymentIntent)
	mux.Get("/api/item/{id}", app.GetItemByID)

	mux.Post("/api/create-customer-and-subscribe-to-plan", app.CreateCustomerAndSubscribe)

	mux.Post("/api/authenticate", app.CreateAuthToken)
	mux.Post("/api/is-authenticated", app.CheckAuthentication)
	mux.Post("/api/forgot-password", app.SendPasswordResetEmail)
	mux.Post("/api/reset-password", app.ResetPassword)

	mux.Route("/api/admin", func(mux chi.Router) {
		mux.Use(app.Auth)

		mux.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("got in"))
		})

		mux.Post("/virtual-terminal-succeeded", app.VirtualTerminalPaymentSuccess)
		mux.Post("/all-sales", app.AllSales)
		mux.Post("/all-subscriptions", app.AllSubscriptions)

		mux.Post("/get-sale/{id}", app.GetSale)
		mux.Post("/refund", app.RefundCharge)
		mux.Post("/cancel-subscription", app.CancelSubscription)

		mux.Post("/all-users", app.AllUsers)
	})

	return mux
}
