package main

import (
  "net/http"
  "sgserver"
  "flag"
)

func main() {
  flag.Parse()
  if flag.NArg() != 2 {
    panic("expected 2 positional arguments: <cert> <key>")
  }
  cert, key := flag.Arg(0), flag.Arg(1)

  rooms := make(map[string]*sgserver.RoomWrapper)
  server := sgserver.NewSuperghostServer(rooms)

  panic(http.ListenAndServeTLS(":443", cert, key, server.Router))
}

