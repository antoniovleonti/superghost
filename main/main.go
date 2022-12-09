package main

import (
  "net/http"
  "sgserver"
)

func main() {
  rooms := make(map[string]*sgserver.RoomWrapper)
  server := sgserver.NewSuperghostServer(rooms)
  http.ListenAndServeTLS(":443", "../crypto/cert.pem", "../crypto/key.pem",
                         server.Router)
}

