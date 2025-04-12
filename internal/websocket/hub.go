package websocket

import (
	"fmt"
	"github.com/qianmianyao/parchment-server/internal/services/chat"
	"github.com/qianmianyao/parchment-server/pkg/global"
)

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	chatCreate *chat.Create
	chatUpdate *chat.Update
	chatFind   *chat.Find
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		chatCreate: chat.NewCreate(),
		chatUpdate: chat.NewUpdate(),
		chatFind:   chat.NewFind(),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clientRegister(client)
		case client := <-h.unregister:
			h.clientUnregister(client)
		case message := <-h.broadcast:
			h.Broadcast(message)
		}
	}
}

// clientRegister registers a new client
func (h *Hub) clientRegister(client *Client) {
	r := h.chatFind.IsUserExist(client.uuid)
	switch r {
	case chat.UserExist:
		if err := h.chatUpdate.UserOnlineStatus(client.uuid, true); err != nil {
			return
		}
	case chat.UserNotExist:
		if err := h.chatCreate.User(client.username, client.uuid); err != nil {
			return
		}
	}
	h.clients[client] = true
	global.Logger.Debug(fmt.Sprintf("client %v connected", client))
}

// clientUnregister unregisters a client
func (h *Hub) clientUnregister(client *Client) {
	if _, ok := h.clients[client]; ok {

		if r := h.chatFind.IsUserExist(client.uuid); r == chat.UserExist {
			if err := h.chatUpdate.UserOnlineStatus(client.uuid, false); err != nil {
				return
			}
		}

		global.Logger.Debug(fmt.Sprintf("client %v disconnected", client))
		delete(h.clients, client)
		close(client.send)
	}
}

// Broadcast sends a message_type to all connected clients
func (h *Hub) Broadcast(message []byte) {
	for client := range h.clients {
		select {
		case client.send <- message:
			global.Logger.Debug(fmt.Sprintf("send to client: %v", client))
		default:
			close(client.send)
			delete(h.clients, client)
		}
	}
}
