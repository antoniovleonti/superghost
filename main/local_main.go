package main

import (
	"net/http"
	"sgserver"
  "os"
  "fmt"
)

func main() {
  if os.Getenv("RAPIDAPI_KEY") == "" {
    panic("environment variable RAPIDAPI_KEY must be set")
  }
	rooms := make(map[string]*sgserver.RoomWrapper)
	server := sgserver.NewSuperghostServer(rooms)
  fmt.Println("Starting server...")
	http.ListenAndServe(":9090", server.Router)
}
