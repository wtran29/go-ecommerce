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
	"github.com/wtran29/go-ecommerce/internal/encryption"
	"github.com/wtran29/go-ecommerce/internal/models"
	"github.com/wtran29/go-ecommerce/internal/urlsigner"
	"golang.org/x/crypto/bcrypt"
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

const (
	Cleared   = 1
	Refunded  = 2
	Cancelled = 3
)

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
			PaymentIntent:       subscription.ID,
			PaymentMethod:       data.PaymentMethod,
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

func (app *application) VirtualTerminalPaymentSuccess(w http.ResponseWriter, r *http.Request) {
	var txnData struct {
		PaymentAmount   int    `json:"amount"`
		PaymentCurrency string `json:"currency"`
		FirstName       string `json:"first_name"`
		LastName        string `json:"last_name"`
		Email           string `json:"email"`
		PaymentIntent   string `json:"payment_intent"`
		PaymentMethod   string `json:"payment_method"`
		BankReturnCode  string `json:"bank_return_code"`
		ExpiryMonth     int    `json:"expiry_month"`
		ExpiryYear      int    `json:"expiry_year"`
		LastFour        string `json:"last_four"`
	}
	err := app.readJSON(w, r, &txnData)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	card := cards.Card{
		Secret: app.config.stripe.secret,
		Key:    app.config.stripe.key,
	}

	pi, err := card.RetrievePaymentIntent(txnData.PaymentIntent)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	pm, err := card.GetPaymentMethod(txnData.PaymentMethod)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	txnData.LastFour = pm.Card.Last4
	txnData.ExpiryMonth = int(pm.Card.ExpMonth)
	txnData.ExpiryYear = int(pm.Card.ExpYear)

	txn := models.Transaction{
		Amount:              txnData.PaymentAmount,
		Currency:            txnData.PaymentCurrency,
		LastFour:            txnData.LastFour,
		ExpiryMonth:         txnData.ExpiryMonth,
		ExpiryYear:          txnData.ExpiryYear,
		PaymentIntent:       txnData.PaymentIntent,
		PaymentMethod:       txnData.PaymentMethod,
		BankReturnCode:      pi.LatestCharge.ID,
		TransactionStatusID: 2,
	}

	_, err = app.SaveTransaction(txn)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	app.writeJSON(w, http.StatusOK, txn)
}

func (app *application) SendPasswordResetEmail(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Email string `json:"email"`
	}

	err := app.readJSON(w, r, &payload)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	// verify email exists
	_, err = app.DB.GetUserByEmail(payload.Email)
	if err != nil {
		var resp struct {
			Error   bool   `json:"error"`
			Message string `json:"message"`
		}
		resp.Error = true
		resp.Message = "No matching email found on our system"
		app.writeJSON(w, http.StatusAccepted, resp)
		return

	}
	link := fmt.Sprintf("%s/reset-password?email=%s", app.config.frontend, payload.Email)
	sign := urlsigner.Signer{
		Secret: []byte(app.config.secretkey),
	}

	signedLink := sign.GenerateTokenFromString(link)

	var data struct {
		Link    string
		Name    string
		Support string
	}

	data.Link = signedLink
	data.Support = "example.com/support"
	data.Name = "Saul Goodman"
	// send mail

	err = app.SendEmail("info@ecomm.com", payload.Email, "Password Reset Request", "password-reset", data)
	if err != nil {
		app.logger.Error(err.Error())
		app.badRequest(w, r, err)
		return
	}
	var resp struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}
	resp.Error = false

	app.writeJSON(w, http.StatusCreated, resp)
}

func (app *application) ResetPassword(w http.ResponseWriter, r *http.Request) {

	var payload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &payload)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	useEncrypt := encryption.Encryption{
		Key: []byte(app.config.secretkey),
	}

	email, err := useEncrypt.Decrypt(payload.Email)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	user, err := app.DB.GetUserByEmail(email)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(payload.Password), 12)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}
	err = app.DB.UpdatePasswordForUser(user, string(newHash))
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	var resp struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}

	resp.Error = false
	resp.Message = "password changed"

	app.writeJSON(w, http.StatusCreated, resp)
}

func (app *application) AllSales(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PageSize    int `json:"page_size"`
		CurrentPage int `json:"page"`
	}
	err := app.readJSON(w, r, &payload)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	isRecurring := false
	allSales, lastPage, totalRecords, err := app.DB.GetAllOrdersPaginated(isRecurring, payload.PageSize, payload.CurrentPage)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	var resp struct {
		CurrentPage  int             `json:"current_page"`
		PageSize     int             `json:"page_size"`
		LastPage     int             `json:"last_page"`
		TotalRecords int             `json:"total_records"`
		Orders       []*models.Order `json:"orders"`
	}
	resp.CurrentPage = payload.CurrentPage
	resp.PageSize = payload.PageSize
	resp.LastPage = lastPage
	resp.TotalRecords = totalRecords
	resp.Orders = allSales

	app.writeJSON(w, http.StatusOK, resp)
}

func (app *application) AllSubscriptions(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PageSize    int `json:"page_size"`
		CurrentPage int `json:"page"`
	}
	err := app.readJSON(w, r, &payload)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	isRecurring := true
	allSubs, lastPage, totalRecords, err := app.DB.GetAllOrdersPaginated(isRecurring, payload.PageSize, payload.CurrentPage)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}
	var resp struct {
		CurrentPage  int             `json:"current_page"`
		PageSize     int             `json:"page_size"`
		LastPage     int             `json:"last_page"`
		TotalRecords int             `json:"total_records"`
		Orders       []*models.Order `json:"orders"`
	}
	resp.CurrentPage = payload.CurrentPage
	resp.PageSize = payload.PageSize
	resp.LastPage = lastPage
	resp.TotalRecords = totalRecords
	resp.Orders = allSubs

	app.writeJSON(w, http.StatusOK, resp)
}

func (app *application) GetSale(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	orderID, err := strconv.Atoi(id)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}
	order, err := app.DB.GetOrderByID(orderID)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	app.writeJSON(w, http.StatusOK, order)
}

func (app *application) RefundCharge(w http.ResponseWriter, r *http.Request) {
	var chargeToRefund struct {
		ID            int    `json:"id"`
		PaymentIntent string `json:"pi"`
		Amount        int    `json:"amount"`
		Currency      string `json:"currency"`
	}

	err := app.readJSON(w, r, &chargeToRefund)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	// validate amount against the order

	card := cards.Card{
		Secret:   app.config.stripe.secret,
		Key:      app.config.stripe.key,
		Currency: chargeToRefund.Currency,
	}

	err = card.Refund(chargeToRefund.PaymentIntent, chargeToRefund.Amount)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}
	// update status in db
	err = app.DB.UpdateOrderStatus(chargeToRefund.ID, Refunded)
	if err != nil {
		app.badRequest(w, r, errors.New("the charge was refunded but database could not be updated"))
		return
	}

	var resp struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}

	resp.Error = false
	resp.Message = "Charge refunded"

	app.writeJSON(w, http.StatusOK, resp)

}

func (app *application) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	var subToCancel struct {
		ID            int    `json:"id"`
		PaymentIntent string `json:"pi"`
		Currency      string `json:"currency"`
	}

	err := app.readJSON(w, r, &subToCancel)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	card := cards.Card{
		Secret:   app.config.stripe.secret,
		Key:      app.config.stripe.key,
		Currency: subToCancel.Currency,
	}

	err = card.CancelSubscription(subToCancel.PaymentIntent)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	// update status in db
	err = app.DB.UpdateOrderStatus(subToCancel.ID, Cancelled)
	if err != nil {
		app.badRequest(w, r, errors.New("the charge was cancelled but database could not be updated"))
		return
	}

	var resp struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}

	resp.Error = false
	resp.Message = "Subscription cancelled"

	app.writeJSON(w, http.StatusOK, resp)
}

func (app *application) AllUsers(w http.ResponseWriter, r *http.Request) {
	allUsers, err := app.DB.GetAllUsers()
	if err != nil {
		app.badRequest(w, r, err)
		return
	}
	app.writeJSON(w, http.StatusOK, allUsers)
}

// OneUser retrieves one user by id through url parse and returns as JSON
func (app *application) OneUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID, err := strconv.Atoi(id)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}
	user, err := app.DB.GetOneUser(userID)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}
	app.writeJSON(w, http.StatusOK, user)
}

func (app *application) EditUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID, err := strconv.Atoi(id)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	var user models.User

	err = app.readJSON(w, r, &user)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}
	if userID > 0 {
		err = app.DB.EditUser(user)
		if err != nil {
			app.badRequest(w, r, err)
			return
		}
		if user.Password != "" {
			newHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), 12)
			if err != nil {
				app.badRequest(w, r, err)
				return
			}
			err = app.DB.UpdatePasswordForUser(user, string(newHash))
			if err != nil {
				app.badRequest(w, r, err)
				return
			}
		}
	} else {
		newHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), 12)
		if err != nil {
			app.badRequest(w, r, err)
			return
		}
		err = app.DB.AddUser(user, string(newHash))
		if err != nil {
			app.badRequest(w, r, err)
			return
		}
	}
	var resp struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}

	resp.Error = false
	app.writeJSON(w, http.StatusOK, resp)
}

func (app *application) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID, err := strconv.Atoi(id)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}
	err = app.DB.DeleteUser(userID)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}
	var resp struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}

	resp.Error = false
	app.writeJSON(w, http.StatusOK, resp)
}
