package main

import (
	"net/http"
)

func (app *application) VirtualTerminal(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "terminal", &templateData{}); err != nil {
		app.logger.Error(err.Error())
	}
}

func (app *application) PaymentReceipt(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	cardHolder := r.Form.Get("cardholder-name")
	email := r.Form.Get("email")

	paymentIntent := r.Form.Get("payment_intent")
	paymentMethod := r.Form.Get("payment_method")
	paymentAmount := r.Form.Get("payment_amount")
	paymentCurrency := r.Form.Get("payment_currency")

	data := make(map[string]interface{})
	data["cardholder"] = cardHolder
	data["email"] = email
	data["pi"] = paymentIntent
	data["pm"] = paymentMethod
	data["pa"] = paymentAmount
	data["pc"] = paymentCurrency

	if err := app.renderTemplate(w, r, "success", &templateData{
		Data: data,
	}); err != nil {
		app.logger.Error(err.Error())
	}

}

// ChargeOneTime displays a template for a one time charge
func (app *application) ChargeOneTime(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "buy-one", nil); err != nil {
		app.logger.Error(err.Error())
	}
}
