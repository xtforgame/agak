package agapiserver

import (
	"fmt"
	// "flag"
	"net/http"
	// "strings"
	// "github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func TestHandleWebsocket(w http.ResponseWriter, r *http.Request) {
	// l := log.WithField("remoteaddr", r.RemoteAddr)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// l.WithError(err).Error("Unable to upgrade connection")
		return
	}

	defer func() {
		conn.Close()
	}()

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			// log.Println(err)
			return
		}
		fmt.Println("p :", p)
		// conn.WriteMessage(websocket.TextMessage, []byte(""))
		// conn.WriteMessage(websocket.BinaryMessage, []byte(""))
		if err := conn.WriteMessage(messageType, p); err != nil {
			// log.Println(err)
			return
		}
	}
}
