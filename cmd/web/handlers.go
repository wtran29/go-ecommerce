package main

import (
	"net/http"

	"github.com/wtran29/go-ecommerce/internal/models"
)

func (app *application) VirtualTerminal(w http.ResponseWriter, r *http.Request) {
	stringMap := make(map[string]string)
	stringMap["publishable_key"] = app.config.stripe.key

	if err := app.renderTemplate(w, r, "terminal", &templateData{
		StringMap: stringMap,
	}, "stripe-js"); err != nil {
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
	item := models.Item{
		ID:             1,
		Name:           "Gopher Plush",
		Description:    "Golang Gopher limited time plush toy.",
		InventoryLevel: 10,
		Price:          1500,
	}
	data := make(map[string]interface{})
	data["item"] = item
	if err := app.renderTemplate(w, r, "buy-one", &templateData{
		Data: data,
	}, "stripe-js"); err != nil {
		app.logger.Error(err.Error())
	}
}
