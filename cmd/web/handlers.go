package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/wtran29/go-ecommerce/internal/cards"
	"github.com/wtran29/go-ecommerce/internal/encryption"
	"github.com/wtran29/go-ecommerce/internal/models"
	"github.com/wtran29/go-ecommerce/internal/urlsigner"
)

// Home displays the home page
func (app *application) Home(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "home", &templateData{}); err != nil {
		app.logger.Error(err.Error())
	}
}

// VirtualTerminal displays terminal page for stripe payment
func (app *application) VirtualTerminal(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "terminal", &templateData{}); err != nil {
		app.logger.Error(err.Error())
	}
}

type TransactionData struct {
	FirstName       string
	LastName        string
	Email           string
	PaymentIntentID string
	PaymentMethodID string
	PaymentAmount   int
	PaymentCurrency string
	LastFour        string
	ExpiryMonth     int
	ExpiryYear      int
	BankReturnCode  string
}

// GetTransactionData gets transaction data from post and stripe
func (app *application) GetTransactionData(r *http.Request) (TransactionData, error) {
	var txnData TransactionData
	err := r.ParseForm()
	if err != nil {
		app.logger.Error(err.Error())
		return txnData, err
	}

	firstName := r.Form.Get("first_name")
	lastName := r.Form.Get("last_name")
	email := r.Form.Get("email")

	paymentIntent := r.Form.Get("payment_intent")
	paymentMethod := r.Form.Get("payment_method")
	paymentAmount := r.Form.Get("payment_amount")
	paymentCurrency := r.Form.Get("payment_currency")

	amount, err := strconv.Atoi(paymentAmount)
	if err != nil {
		app.logger.Error(err.Error())
		return txnData, err
	}

	card := cards.Card{
		Secret: app.config.stripe.secret,
		Key:    app.config.stripe.key,
	}

	pi, err := card.RetrievePaymentIntent(paymentIntent)
	if err != nil {
		app.logger.Error(err.Error())
		return txnData, err
	}

	pm, err := card.GetPaymentMethod(paymentMethod)
	if err != nil {
		app.logger.Error(err.Error())
		return txnData, err
	}

	lastFour := pm.Card.Last4
	expiryMonth := pm.Card.ExpMonth
	expiryYear := pm.Card.ExpYear

	txnData = TransactionData{
		FirstName:       firstName,
		LastName:        lastName,
		Email:           email,
		PaymentIntentID: paymentIntent,
		PaymentMethodID: paymentMethod,
		PaymentAmount:   amount,
		PaymentCurrency: paymentCurrency,
		LastFour:        lastFour,
		ExpiryMonth:     int(expiryMonth),
		ExpiryYear:      int(expiryYear),
		BankReturnCode:  pi.LatestCharge.ID,
	}
	return txnData, nil

}

// PaymentSuccess displays the receipt page
func (app *application) PaymentSuccess(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	itemID, err := strconv.Atoi(r.Form.Get("product_id"))
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	txnData, err := app.GetTransactionData(r)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	// Create new customer
	customer := models.Customer{
		FirstName: txnData.FirstName,
		LastName:  txnData.LastName,
		Email:     txnData.Email,
	}

	customerID, err := app.SaveCustomer(customer)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	// create a new transaction
	txn := models.Transaction{
		Amount:              txnData.PaymentAmount,
		Currency:            txnData.PaymentCurrency,
		LastFour:            txnData.LastFour,
		ExpiryMonth:         txnData.ExpiryMonth,
		ExpiryYear:          txnData.ExpiryYear,
		BankReturnCode:      txnData.BankReturnCode,
		PaymentIntent:       txnData.PaymentIntentID,
		PaymentMethod:       txnData.PaymentMethodID,
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
		Amount:        txnData.PaymentAmount,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	_, err = app.SaveOrder(order)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	// write data to session, redirect to new page
	app.Session.Put(r.Context(), "receipt", txnData)
	http.Redirect(w, r, "/receipt", http.StatusSeeOther)
}

// VirtualTerminalPaymentSuccess displays the receipt page for virtual terminal transactions
func (app *application) VirtualTerminalPaymentSuccess(w http.ResponseWriter, r *http.Request) {

	txnData, err := app.GetTransactionData(r)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	// create a new transaction
	txn := models.Transaction{
		Amount:              txnData.PaymentAmount,
		Currency:            txnData.PaymentCurrency,
		LastFour:            txnData.LastFour,
		ExpiryMonth:         txnData.ExpiryMonth,
		ExpiryYear:          txnData.ExpiryYear,
		BankReturnCode:      txnData.BankReturnCode,
		PaymentIntent:       txnData.PaymentIntentID,
		PaymentMethod:       txnData.PaymentMethodID,
		TransactionStatusID: 2,
	}

	_, err = app.SaveTransaction(txn)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	// write data to session, redirect to new page
	app.Session.Put(r.Context(), "receipt", txnData)
	http.Redirect(w, r, "/virtual-terminal-receipt", http.StatusSeeOther)

}

func (app *application) Receipt(w http.ResponseWriter, r *http.Request) {
	txn := app.Session.Get(r.Context(), "receipt").(TransactionData)
	data := make(map[string]interface{})
	data["txn"] = txn
	app.Session.Remove(r.Context(), "receipt")
	if err := app.renderTemplate(w, r, "receipt", &templateData{
		Data: data,
	}); err != nil {
		app.logger.Error(err.Error())
	}
}

func (app *application) VirtualTerminalReceipt(w http.ResponseWriter, r *http.Request) {
	txn := app.Session.Get(r.Context(), "receipt").(TransactionData)
	data := make(map[string]interface{})
	data["txn"] = txn
	app.Session.Remove(r.Context(), "receipt")
	if err := app.renderTemplate(w, r, "virtual-terminal-receipt", &templateData{
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

func (app *application) BronzePlan(w http.ResponseWriter, r *http.Request) {
	item, err := app.DB.GetItem(2)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}
	data := make(map[string]interface{})
	data["item"] = item
	if err := app.renderTemplate(w, r, "bronze-plan", &templateData{
		Data: data,
	}); err != nil {
		app.logger.Error(err.Error())
	}
}

func (app *application) BronzePlanReceipt(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "receipt-plan", &templateData{}); err != nil {
		app.logger.Error(err.Error())
	}
}

// LoginPage displays login page
func (app *application) LoginPage(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "login", &templateData{}); err != nil {
		app.logger.Error(err.Error())
	}
}

func (app *application) PostLoginPage(w http.ResponseWriter, r *http.Request) {
	app.Session.RenewToken(r.Context())

	err := r.ParseForm()
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	email := r.Form.Get("email")
	password := r.Form.Get("password")

	id, err := app.DB.Authenticate(email, password)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	app.Session.Put(r.Context(), "userID", id)
	http.Redirect(w, r, "/", http.StatusSeeOther)

}

func (app *application) Logout(w http.ResponseWriter, r *http.Request) {
	app.Session.Destroy(r.Context())
	app.Session.RenewToken(r.Context())
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
func (app *application) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "forgot-password", &templateData{}); err != nil {
		app.logger.Error(err.Error())
	}
}

func (app *application) ShowResetPassword(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	theURL := r.RequestURI
	testURL := fmt.Sprintf("%s%s", app.config.frontend, theURL)

	signer := urlsigner.Signer{
		Secret: []byte(app.config.secretkey),
	}

	valid := signer.VerifyToken(testURL)
	if !valid {
		app.logger.Error("Invalid url tampering detected")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	// check reset password expiry
	expired := signer.Expired(testURL, 60)
	if expired {
		app.logger.Error("Password reset link expired!")
		// redirect to error page
		http.Error(w, "Password reset link expired! Please request a new password reset.", http.StatusBadRequest)
		return
	}

	useEncrypt := encryption.Encryption{
		Key: []byte(app.config.secretkey),
	}

	encryptedEmail, err := useEncrypt.Encrypt(email)
	if err != nil {
		app.logger.Error("Encryption failed!")
		return
	}

	data := make(map[string]interface{})
	data["email"] = encryptedEmail

	if err := app.renderTemplate(w, r, "reset-password", &templateData{
		Data: data,
	}); err != nil {
		app.logger.Error(err.Error())
	}

}

func (app *application) AllSales(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "all-sales", &templateData{}); err != nil {
		app.logger.Error(err.Error())
	}
}

func (app *application) AllSubscriptions(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "all-subscriptions", &templateData{}); err != nil {
		app.logger.Error(err.Error())
	}
}
func (app *application) ShowSale(w http.ResponseWriter, r *http.Request) {
	stringMap := make(map[string]string)
	stringMap["title"] = "Sale"
	stringMap["cancel"] = "/admin/all-sales"
	stringMap["refund-url"] = "/api/admin/refund"
	stringMap["refund-btn"] = "Refund Order"
	stringMap["messages"] = "Charge refunded"
	stringMap["text"] = "Your charge has been refunded."
	stringMap["status"] = "Refunded"
	stringMap["bg-status"] = "bg-danger"
	if err := app.renderTemplate(w, r, "sale", &templateData{
		StringMap: stringMap,
	}); err != nil {
		app.logger.Error(err.Error())
	}
}

func (app *application) ShowSubscription(w http.ResponseWriter, r *http.Request) {
	stringMap := make(map[string]string)
	stringMap["title"] = "Subscription"
	stringMap["cancel"] = "/admin/all-subscriptions"
	stringMap["refund-url"] = "/api/admin/cancel-subscription"
	stringMap["refund-btn"] = "Cancel Subscription"
	stringMap["messages"] = "Subscription Cancelled"
	stringMap["text"] = "Your subscription has been cancelled."
	stringMap["status"] = "Cancelled"
	stringMap["bg-status"] = "bg-dark"

	if err := app.renderTemplate(w, r, "sale", &templateData{
		StringMap: stringMap,
	}); err != nil {
		app.logger.Error(err.Error())
	}
}

// AllUsers shows the all users page
func (app *application) AllUsers(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "all-users", &templateData{}); err != nil {
		app.logger.Error(err.Error())
	}
}

// OneUser shows one admin user for add/edit/delete
func (app *application) OneUser(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "one-user", &templateData{}); err != nil {
		app.logger.Error(err.Error())
	}
}
