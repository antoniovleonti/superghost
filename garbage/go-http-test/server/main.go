package main

import (
  "errors"
  "fmt"
  "github.com/gorilla/mux"
  "net/http"
  "os"
  // "encoding/json"
  // "io"
  // "strconv"
  // "strings"
  // "sync"
)

func main() {
  gm := GamesManager{stringToGame: make(map[string]*Game)}
  gamesHandler := GamesHandler{Manager: gm}
  gameHandler := GameHandler{Manager: gm}
  wordHandler := WordHandler{Manager: gm}
  playersHandler := PlayersHandler{Manager: gm}
  playerHandler := PlayerHandler{Manager: gm}

  router := mux.NewRouter()
  // api:= router.PathPrefix("/api/v0").Subrouter()

  // TODO: HEAD, OPTIONS
  router.HandleFunc("/games", gamesHandler.Handle).Methods(
      http.MethodGet, http.MethodPost)

  // TODO: HEAD, OPTIONS
  router.HandleFunc("/games/{game}", gameHandler.Handle).Methods(
      http.MethodHead, http.MethodOptions, http.MethodGet)

  // TODO: HEAD, OPTIONS
  router.HandleFunc("/games/{game}/word", wordHandler.Handle).Methods(
      http.MethodHead, http.MethodOptions, http.MethodGet, http.MethodPost)

  // TODO: HEAD, OPTIONS, POST
  router.HandleFunc("/games/{game}/challenge-no-word", wordHandler.Handle).
      Methods(http.MethodHead, http.MethodOptions, http.MethodPost)

  // TODO: HEAD, OPTIONS, POST
  router.HandleFunc("/games/{game}/challenge-is-word", wordHandler.Handle).Methods(
      http.MethodHead, http.MethodOptions, http.MethodPost)

  // TODO: HEAD, OPTIONS, POST
  router.HandleFunc("/games/{game}/concede", ConcedeHandler).
      Methods(http.MethodHead, http.MethodOptions, http.MethodPost)

  // TODO: HEAD, OPTIONS
  router.HandleFunc("/games/{game}/players", playersHandler.Handle).Methods(
      http.MethodHead, http.MethodOptions, http.MethodGet, http.MethodPost)

  // TODO: HEAD, OPTIONS
  router.HandleFunc("/games/{game}/players/{player}", playerHandler.Handle).
      Methods(http.MethodHead, http.MethodOptions, http.MethodGet)

  err := http.ListenAndServe(":3333", router)
  if errors.Is(err, http.ErrServerClosed) {
    fmt.Printf("Server closed.\n")
  } else if err != nil {
    fmt.Printf("Error starting server: %s\n", err)
		os.Exit(1)
  }
}
