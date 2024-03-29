package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

type WebSocketConnection struct {
	*websocket.Conn
}

type WsPayload struct {
	Action      string              `json:"action"`
	Message     string              `json:"message"`
	Username    string              `json:"username"`
	MessageType string              `json:"message_type"`
	UserID      int                 `json:"user_id"`
	Conn        WebSocketConnection `json:"-"`
}

type WsJsonResponse struct {
	Action  string `json:"action"`
	Message string `json:"message"`
	UserID  int    `json:"user_id"`
}

var upgradeConnection = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

var clients = make(map[WebSocketConnection]string)

var wsChan = make(chan WsPayload)

func (app *application) WsEndPoint(w http.ResponseWriter, r *http.Request) {
	wsUpgrade := w
	if upgrade, ok := w.(interface{ Unwrap() http.ResponseWriter }); ok {
		wsUpgrade = upgrade.Unwrap()
	}
	app.logger.Debug(fmt.Sprintf("w's type is %T\n", w))
	ws, err := upgradeConnection.Upgrade(wsUpgrade, r, nil)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	app.logger.Info(fmt.Sprintf("Client connected from %s", r.RemoteAddr))
	var response WsJsonResponse
	response.Message = "Connected to server"

	err = ws.WriteJSON(response)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	conn := WebSocketConnection{Conn: ws}
	clients[conn] = ""

	go app.ListenForWS(&conn)
}

func (app *application) ListenForWS(conn *WebSocketConnection) {
	defer func() {
		if r := recover(); r != nil {
			app.logger.Error(fmt.Sprintf("%v", r))
		}
	}()

	var payload WsPayload
	for {
		err := conn.ReadJSON(&payload)
		if err != nil {
			// do nothing
			_ = conn.Close()
			delete(clients, *conn)
			break
		} else {
			payload.Conn = *conn
			app.logger.Info(fmt.Sprintf("ListenForWS: %v", payload.Action))
			wsChan <- payload
		}

	}
}

func (app *application) ListenToWsChannel() {
	var response WsJsonResponse
	for {
		e := <-wsChan
		switch e.Action {
		case "deleteUser":
			response.Action = "logout"
			response.Message = "Your account has been deleted."
			response.UserID = e.UserID
			app.broadcastToAll(response)

		default:
		}
	}
}

func (app *application) broadcastToAll(response WsJsonResponse) {
	for client := range clients {
		// broadcast to all connected client
		err := client.WriteJSON(response)
		if err != nil {
			app.logger.Error(fmt.Sprintf("Websocket err on %s: %s", response.Action, err))
			_ = client.Close()
			delete(clients, client)
		}
	}
}
