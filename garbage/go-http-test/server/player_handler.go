package main

import (
  "encoding/json"
  "fmt"
  "github.com/gorilla/mux"
  "net/http"
)

type PlayerHandler struct {
  Manager GamesManager
}

func (h *PlayerHandler) Handle(w http.ResponseWriter, r *http.Request) {
  gameStr := mux.Vars(r)["game"]
  playerStr := mux.Vars(r)["player"]
  // first check to make sure game is valid
  game, ok := h.Manager.GetGame(gameStr)
  if !ok {
    http.NotFound(w, r)
    return
  }
  // now the same for the player
  player, ok := game.GetPlayer(playerStr)
  if !ok {
    http.NotFound(w, r)
    return
  }

  switch r.Method {
    case http.MethodHead:
      h.head(player, w, r)
    case http.MethodOptions:
      h.options(player, w, r)
    case http.MethodGet:
      h.get(player, w, r)
    default:
      // should not be possible to get here
  }
}

func (h *PlayerHandler) head(player *Player, w http.ResponseWriter,
                             r *http.Request) {}
func (h *PlayerHandler) options(player *Player, w http.ResponseWriter,
                                r *http.Request) {}
func (h *PlayerHandler) get(player *Player, w http.ResponseWriter,
                            r *http.Request) {
  playerJson, err := json.Marshal(player)
  if (err != nil) {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  w.Header().Set("Content-Type", "application/json; charset=utf-8")
  fmt.Fprintln(w, string(playerJson))
}
