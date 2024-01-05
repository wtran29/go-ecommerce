package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stripe/stripe-go/v76"
	"github.com/wtran29/go-ecommerce/internal/cards"
	"github.com/wtran29/go-ecommerce/internal/models"
)

type stripePayload struct {
	Currency      string `json:"currency"`
	Amount        string `json:"amount"`
	PaymentMethod string `json:"payment_method"`
	Email         string `json:"email"`
	Cardbrand     string `json:"card_brand"`
	ExpiryMonth   int    `json:"exp_month"`
	ExpiryYear    int    `json:"exp_year"`
	LastFour      string `json:"last_four"`
	Plan          string `json:"plan"`
	ProductID     string `json:"product_id"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
}

type jsonResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Content string `json:"content,omitempty"`
	ID      int    `json:"id,omitempty"`
}

func (app *application) GetPaymentIntent(w http.ResponseWriter, r *http.Request) {
	var payload stripePayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	amount, err := strconv.Atoi(payload.Amount)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	card := cards.Card{
		Secret:   app.config.stripe.secret,
		Key:      app.config.stripe.key,
		Currency: payload.Currency,
	}

	isValid := true

	pi, msg, err := card.Charge(payload.Currency, amount)
	if err != nil {
		isValid = false
	}

	if isValid {
		out, err := json.MarshalIndent(pi, "", "  ")
		if err != nil {
			app.logger.Error(err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
	} else {
		j := jsonResponse{
			OK:      false,
			Message: msg,
			Content: "",
		}

		out, err := json.MarshalIndent(j, "", "  ")
		if err != nil {
			app.logger.Error(err.Error())
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)

	}

}

func (app *application) GetItemByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	itemID, _ := strconv.Atoi(id)

	item, err := app.DB.GetItem(itemID)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}
	out, err := json.MarshalIndent(item, "", " ")
	if err != nil {
		app.logger.Error(err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

func (app *application) CreateCustomerAndSubscribe(w http.ResponseWriter, r *http.Request) {
	var data stripePayload
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		app.logger.Error(err.Error())
		http.Error(w, "Error decoding JSON", http.StatusBadRequest)
		return
	}

	app.logger.Info(fmt.Sprintf("data: %v %v %v %v", data.Email, data.LastFour, data.PaymentMethod, data.Plan))

	card := cards.Card{
		Secret:   app.config.stripe.secret,
		Key:      app.config.stripe.key,
		Currency: data.Currency,
	}

	okay := true
	var subscription *stripe.Subscription
	txnMsg := "Transaction successful"

	stripeCustomer, msg, err := card.CreateCustomer(data.PaymentMethod, data.Email)
	if err != nil {
		app.logger.Error(err.Error())
		okay = false
		txnMsg = msg
	}

	if okay {
		subscription, err = card.SubscribeToPlan(stripeCustomer, data.Plan, data.Email, data.LastFour, "")
		if err != nil {
			app.logger.Error(err.Error())
			okay = false
			txnMsg = "Error subscribing customer"
			return
		}

		app.logger.Info(fmt.Sprintf("subscription id is %v", subscription.ID))
	}

	if okay {
		productID, _ := strconv.Atoi(data.ProductID)
		customer := models.Customer{
			FirstName: data.FirstName,
			LastName:  data.LastName,
			Email:     data.Email,
		}
		customerID, err := app.SaveCustomer(customer)
		if err != nil {
			app.logger.Error(err.Error())
			return
		}

		amount, _ := strconv.Atoi(data.Amount)
		// expiryMonth, _ := strconv.Atoi(data.ExpiryMonth)
		// expiryYear, _ := strconv.Atoi(data.ExpiryYear)
		txn := models.Transaction{
			Amount:              amount,
			Currency:            "usd",
			LastFour:            data.LastFour,
			ExpiryMonth:         data.ExpiryMonth,
			ExpiryYear:          data.ExpiryYear,
			TransactionStatusID: 2,
		}

		txnID, err := app.SaveTransaction(txn)
		if err != nil {
			app.logger.Error(err.Error())
			return
		}

		orderID := models.Order{
			ItemID:        productID,
			TransactionID: txnID,
			CustomerID:    customerID,
			StatusID:      1,
			Quantity:      1,
			Amount:        amount,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		_, err = app.SaveOrder(orderID)
		if err != nil {
			app.logger.Error(err.Error())
			return
		}

	}
	// msg := ""

	resp := jsonResponse{
		OK:      okay,
		Message: txnMsg,
	}

	out, err := json.MarshalIndent(resp, "", " ")
	if err != nil {
		app.logger.Error(err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")

	w.Write(out)
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

// CreateAuthToken creates an auth token
func (app *application) CreateAuthToken(w http.ResponseWriter, r *http.Request) {
	var userInput struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &userInput)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	// get user by email in db
	user, err := app.DB.GetUserByEmail(userInput.Email)
	if err != nil {
		app.InvalidCredentials(w)
		return
	}
	// validate password
	validPassword, err := app.PasswordMatches(user.Password, userInput.Password)
	if err != nil {
		app.InvalidCredentials(w)
		return
	}

	if !validPassword {
		app.InvalidCredentials(w)
		return
	}
	// generate token
	token, err := models.GenerateToken(user.ID, 24*time.Hour, models.ScopeAuthentication)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	// save to db
	err = app.DB.InsertToken(token, user)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	// send response
	var payload struct {
		Error   bool          `json:"error"`
		Message string        `json:"message"`
		Token   *models.Token `json:"access_token"`
	}

	payload.Error = false
	payload.Message = fmt.Sprintf("token for %s created", userInput.Email)
	payload.Token = token

	_ = app.writeJSON(w, http.StatusOK, payload)
}

func (app *application) authenticateToken(r *http.Request) (*models.User, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, errors.New("no authorization header received")
	}

	headerParts := strings.Split(authHeader, " ")
	if len(headerParts) != 2 || headerParts[0] != "Bearer" {
		return nil, errors.New("no authorization header received")
	}

	token := headerParts[1]
	if len(token) != 26 {
		return nil, errors.New("wrong authentication token size")
	}

	// get the user from the tokens table
	user, err := app.DB.GetUserForToken(token)
	if err != nil {
		return nil, errors.New("no matching user found")
	}

	return user, nil
}

func (app *application) CheckAuthentication(w http.ResponseWriter, r *http.Request) {
	// validate token and get user
	user, err := app.authenticateToken(r)
	if err != nil {
		app.InvalidCredentials(w)
		return
	}

	var payload struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}
	payload.Error = false
	payload.Message = fmt.Sprintf("authenicated user %s", user.Email)
	app.writeJSON(w, http.StatusOK, payload)
}
