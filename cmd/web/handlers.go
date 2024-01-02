package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/wtran29/go-ecommerce/internal/cards"
	"github.com/wtran29/go-ecommerce/internal/models"
)

// Home displays the home page
func (app *application) Home(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "home", &templateData{}); err != nil {
		app.logger.Error(err.Error())
	}
}

// VirtualTerminal displays terminal page for stripe payment
func (app *application) VirtualTerminal(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "terminal", &templateData{}, "stripe-js"); err != nil {
		app.logger.Error(err.Error())
	}
}

func (app *application) PaymentReceipt(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	firstName := r.Form.Get("first_name")
	lastName := r.Form.Get("last_name")
	email := r.Form.Get("email")

	paymentIntent := r.Form.Get("payment_intent")
	paymentMethod := r.Form.Get("payment_method")
	paymentAmount := r.Form.Get("payment_amount")
	paymentCurrency := r.Form.Get("payment_currency")

	itemID, err := strconv.Atoi(r.Form.Get("product_id"))
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	card := cards.Card{
		Secret: app.config.stripe.secret,
		Key:    app.config.stripe.key,
	}

	pi, err := card.RetrievePaymentIntent(paymentIntent)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	pm, err := card.GetPaymentMethod(paymentMethod)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	lastFour := pm.Card.Last4
	expiryMonth := pm.Card.ExpMonth
	expiryYear := pm.Card.ExpYear

	// Create new customer
	customer := models.Customer{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
	}

	customerID, err := app.SaveCustomer(customer)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	// Create new transaction
	amount, err := strconv.Atoi(paymentAmount)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}
	txn := models.Transaction{
		Amount:              amount,
		Currency:            paymentCurrency,
		LastFour:            lastFour,
		ExpiryMonth:         int(expiryMonth),
		ExpiryYear:          int(expiryYear),
		BankReturnCode:      pi.LatestCharge.ID,
		TransactionStatusID: 2,
	}

	txnID, err := app.SaveTransaction(txn)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	// Create new order
	order := models.Order{
		ItemID:        itemID,
		TransactionID: txnID,
		CustomerID:    customerID,
		StatusID:      1,
		Quantity:      1,
		Amount:        amount,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	_, err = app.SaveOrder(order)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	data := make(map[string]interface{})
	data["first_name"] = firstName
	data["last_name"] = lastName
	data["email"] = email
	data["pi"] = paymentIntent
	data["pm"] = paymentMethod
	data["pa"] = paymentAmount
	data["pc"] = paymentCurrency
	data["last_four"] = lastFour
	data["expiry_month"] = expiryMonth
	data["expiry_year"] = expiryYear
	data["bank_return_code"] = pi.LatestCharge.ID

	// write data to session, redirect to new page

	if err := app.renderTemplate(w, r, "success", &templateData{
		Data: data,
	}); err != nil {
		app.logger.Error(err.Error())
	}

}

// SaveCustomer saves a customer and returns customer id
func (app *application) SaveCustomer(customer models.Customer) (int, error) {
	id, err := app.DB.InsertCustomer(customer)
	if err != nil {
		app.logger.Error(err.Error())
		return 0, err
	}
	return id, nil
}

// SaveTransaction saves a transaction and returns txn id
func (app *application) SaveTransaction(txn models.Transaction) (int, error) {
	id, err := app.DB.InsertTransaction(txn)
	if err != nil {
		app.logger.Error(err.Error())
		return 0, err
	}
	return id, nil
}

// SaveOrder saves a order and returns order id
func (app *application) SaveOrder(order models.Order) (int, error) {
	id, err := app.DB.InsertOrder(order)
	if err != nil {
		app.logger.Error(err.Error())
		return 0, err
	}
	return id, nil
}

// ChargeOneTime displays a template for a one time charge
func (app *application) ChargeOneTime(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	itemID, _ := strconv.Atoi(id)

	item, err := app.DB.GetItem(itemID)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	data := make(map[string]interface{})
	data["item"] = item
	if err := app.renderTemplate(w, r, "buy-one", &templateData{
		Data: data,
	}, "stripe-js"); err != nil {
		app.logger.Error(err.Error())
	}
}
