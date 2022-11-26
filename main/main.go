package main

import (
  "net/http"
  "sgserver"
)

func main() {
  server := sgserver.NewSuperghostServer()
  http.ListenAndServe(":9090", server.Router)
}

