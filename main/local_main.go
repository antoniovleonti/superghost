package main

import (
	"net/http"
	"sgserver"
)

func main() {
	rooms := make(map[string]*sgserver.RoomWrapper)
	server := sgserver.NewSuperghostServer(rooms)
	http.ListenAndServe(":9090", server.Router)
}
