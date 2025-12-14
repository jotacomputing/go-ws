package ws

import (
	"encoding/json"
	contracts "exchange/Contracts"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{}

type ClientMessage struct {
	Socket  *websocket.Conn // connection objexct needs to be sent along with the message
	Payload contracts.MessageFromUser
}
type Server struct{
	subscriber contracts.SubscriberForWs
	unsubscriber contracts.UnSubscriberForWs 
	cleanup contracts.CleanUpForWs
}

func NewServer(
    sub contracts.SubscriberForWs,
    unsub contracts.UnSubscriberForWs,
    cleanup contracts.CleanUpForWs,
) *Server {
    return &Server{
        subscriber:   sub,
        unsubscriber: unsub,
        cleanup:      cleanup,
    }
}


func (s *Server)wsHandler(c echo.Context) error {

	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)

	if err != nil {
		fmt.Println("UPGRADE ERROR:", err)
		return err
	}
	defer func(){
		ws.Close()
		s.cleanup.CleanupConnection(ws)
	}()

	var mess contracts.MessageFromUser
	//fmt.Println("WebSocket connection established!")

	for {
		_, p, err := ws.ReadMessage()
		if err != nil {
			fmt.Println("READ ERROR:", err)
			return nil
		}
		if err := json.Unmarshal(p, &mess); err != nil {
			fmt.Println("json error:", err)
			continue
		}
		fmt.Println("Recived message")
		switch mess.Method{
		case contracts.SUBSCRIBE:
			if len(mess.Params) > 0 {
				s.subscriber.Subscribe(mess.Params[0] , ws)
			}

		case contracts.UNSUBSCRIBE :
			if len(mess.Params) > 0 {
				s.unsubscriber.UnSubscribe(mess.Params[0] , ws)
			}
		}

	}
}

func(s * Server) CreateServer() {
	fmt.Println("BOOTING SERVER...")

	e := echo.New()
	e.GET("/ws", s.wsHandler)

	fmt.Println("LISTENING on :8080 ...")

	err := e.Start(":8080")
	fmt.Println("SERVER EXITED:", err)
}
