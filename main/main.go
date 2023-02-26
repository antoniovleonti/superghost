package main

import (
  "flag"
  "fmt"
  "net/http"
  "os"
  "sgserver"
)

func main() {
  flag.Parse()
  if flag.NArg() != 2 {
    panic("expected 2 positional arguments: <cert> <key>")
  }
  cert, key := flag.Arg(0), flag.Arg(1)
  if os.Getenv("RAPIDAPI_KEY") == "" {
    panic("environment variable RAPIDAPI_KEY must be set")
  }

  rooms := make(map[string]*sgserver.RoomWrapper)
  server := sgserver.NewSuperghostServer(rooms)

  fmt.Println("Starting server...")
  panic(http.ListenAndServeTLS(":443", cert, key, server.Router))
}

