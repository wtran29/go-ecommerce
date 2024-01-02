package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/wtran29/go-ecommerce/internal/cards"
)

type stripePayload struct {
	Currency string `json:"currency"`
	Amount   string `json:"amount"`
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
