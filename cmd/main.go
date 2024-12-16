package main

import (
	"fmt"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"golang.org/x/net/websocket"
)

type ChatRoom struct {
	Name    string
	Clients map[*websocket.Conn]string
	mu      sync.Mutex
}

var (
	chatRooms = make(map[string]*ChatRoom)
)

func getOrCreateRoom(roomName string) *ChatRoom {
	if room, exists := chatRooms[roomName]; exists {
		return room
	}

	room := &ChatRoom{
		Name:    roomName,
		Clients: make(map[*websocket.Conn]string),
	}
	chatRooms[roomName] = room
	return room
}

func broadcastMessage(room *ChatRoom, message string) {
	room.mu.Lock()
	defer room.mu.Unlock()

	for client := range room.Clients {
		err := websocket.Message.Send(client, message)
		if err != nil {
			log.Printf("Error sending message: %s\n", err)
		}
	}
}

func chat(c echo.Context) error {
	roomName := c.QueryParam("room")
	userName := c.QueryParam("user")

	if roomName == "" || userName == "" {
		return c.String(400, "room and user query parameters are required")
	}

	room := getOrCreateRoom(roomName)

	websocket.Handler(func(ws *websocket.Conn) {
		defer func() {
			room.mu.Lock()
			delete(room.Clients, ws)
			room.mu.Unlock()
			ws.Close()
			log.Printf("%s disconnected from room: %s\n", userName, roomName)
		}()

		room.mu.Lock()
		room.Clients[ws] = userName
		room.mu.Unlock()

		for {
			var msg string
			err := websocket.Message.Receive(ws, &msg)
			if err != nil {
				log.Printf("Error receiving message: %v\n", err)
				break
			}
			log.Printf("Received message from %s in room %s: %s\n", userName, roomName, msg)
			broadcastMessage(room, fmt.Sprintf("%s: %s", userName, msg))
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
    e.Use(middleware.CORS())
    e.Static("/", "./../public")

	e.GET("/ws", chat)
	e.Logger.Fatal(e.Start(":3030"))
}
