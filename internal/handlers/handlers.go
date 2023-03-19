package handlers

import (
	"fmt"
	"github.com/CloudyKit/jet/v6"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sort"
)

// wsChan is the channel for the WsPayload's of the events
var wsChan = make(chan WsPayload)

// clients holds the clients that are connected to the web socket
var clients = make(map[WebSocketConn]string)

// views holds the templates
var views = jet.NewSet(
	jet.NewOSFileSystemLoader("./html"),
	jet.InDevelopmentMode(),
)

// WebSocketConn holds the Web Socket connection
type WebSocketConn struct {
	*websocket.Conn
}

// WsJsonResponse holds the response data
type WsJsonResponse struct {
	Action         string   `json:"action"`
	Message        string   `json:"message"`
	MessageType    string   `json:"messageType"`
	ConnectedUsers []string `json:"connectedUsers"`
}

// WsPayload holds the data for the payload
type WsPayload struct {
	Action   string        `json:"action"`
	Username string        `json:"username"`
	Message  string        `json:"message"`
	Conn     WebSocketConn `json:"-"`
}

var upgradeConnection = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Home renders the home page
func Home(w http.ResponseWriter, _ *http.Request) {
	err := renderPage(w, "home", nil)
	if err != nil {
		log.Fatal(err)
	}
	return
}

// WsEndpoint upgrades connection to websocket
func WsEndpoint(w http.ResponseWriter, r *http.Request) {
	ws, err := upgradeConnection.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	log.Println("Client connected to endpoint")
	response := WsJsonResponse{
		Message: `
			<em><small>Connected to server</small></em>
		`,
	}

	conn := WebSocketConn{Conn: ws}
	clients[conn] = ""

	err = ws.WriteJSON(&response)
	if err != nil {
		log.Println(err)
	}

	go ListenForWs(&conn)
}

// ListenForWs keeps the WebSocket Connection alive even if it crashes
func ListenForWs(conn *WebSocketConn) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Error", fmt.Sprintf("%v", r))
		}
	}()

	var payload WsPayload

	for {
		err := conn.ReadJSON(&payload)
		if err != nil {
			// do nothing
		} else {
			payload.Conn = *conn
			wsChan <- payload
		}
	}
}

// ListenToWsChannel listens for the wsChan if any value passed it broadcasts to all
func ListenToWsChannel() {
	var response WsJsonResponse

	for {
		// once some values passed into the channel we put it in a new event
		event := <-wsChan

		switch event.Action {
		case "username":
			clients[event.Conn] = event.Username
			users := getUsers()
			response.Action = "usersUpdate"
			response.ConnectedUsers = users
			response.Message = fmt.Sprintf("<strong>%s</strong> joined <br/>", event.Username)
			broadcastToAll(response)
		case "left":
			response.Action = "usersUpdate"
			delete(clients, event.Conn)
			users := getUsers()
			response.ConnectedUsers = users
			broadcastToAll(response)
		case "message":
			response.Action = "messagesUpdate"
			response.ConnectedUsers = getUsers()
			response.Message = fmt.Sprintf("<strong>%s</strong> : %s <br/>", event.Username, event.Message)
			broadcastToAll(response)
		}

		// then we broadcast it to all users in the chat
		broadcastToAll(response)
	}
}

// getUserList gets all the usernames
func getUsers() []string {
	var users []string
	for _, v := range clients {
		if v != "" {
			users = append(users, v)
		}
	}
	sort.Strings(users)
	return users
}

// broadcastToAll sends a response when an event occurs
func broadcastToAll(r WsJsonResponse) {
	// here we loop through the clients available
	for client := range clients {
		// here we send the response to the client
		err := client.WriteJSON(r)
		// if any error returns we log it
		if err != nil {
			log.Println("Websocket error: ", err)
			// then we close the websocket connection and delete the user from the clients map
			_ = client.Close()
			delete(clients, client)
		}
	}
}

// renderPage takes a responseWriter, template name and data and renders the template
func renderPage(w http.ResponseWriter, tmpl string, data jet.VarMap) error {
	view, err := views.GetTemplate(tmpl)
	if err != nil {
		log.Fatal(err)
		return err
	}

	err = view.Execute(w, data, nil)

	if err != nil {
		log.Fatal(err)
		return err
	}

	return nil
}
