package main

import (
  "encoding/json"
  "fmt"
  "github.com/gorilla/mux"
  "net/http"
)

type GameHandler struct {
  Manager GamesManager
}

func (h *GameHandler) Handle(w http.ResponseWriter, r *http.Request) {
  gameStr := mux.Vars(r)["game"]
  game, ok := h.Manager.GetGame(gameStr);
  // first check to make sure game is valid
  if !ok {
    http.NotFound(w, r)
    return
  }

  switch r.Method {
    case http.MethodHead:
      h.head(game, w, r)
    case http.MethodOptions:
      h.options(game, w, r)
    case http.MethodGet:
      h.get(game, w, r)
    default:
      // Shouldn't be possible to get here
  }
}

func (h *GameHandler) head(game *Game, w http.ResponseWriter, r *http.Request) {
}

func (h *GameHandler) options(game *Game, w http.ResponseWriter,
                              r *http.Request) {
}

func (h *GameHandler) get(game *Game, w http.ResponseWriter, r *http.Request) {
  b, err := json.Marshal(game)
  if (err != nil) {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  w.Header().Set("Content-Type", "application/json; charset=utf-8")
  fmt.Fprintln(w, string(b))
}
