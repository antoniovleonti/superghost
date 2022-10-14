package main

import (
  "encoding/json"
  "fmt"
  "github.com/gorilla/mux"
  "net/http"
)

type PlayersHandler struct {
  Manager GamesManager
}

// Handlers for /games/{game}/players
func (h *PlayersHandler) Handle(w http.ResponseWriter, r *http.Request) {
  // validate game
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
    case http.MethodPost:
      h.post(game, w, r)
    default:
      // Shouldn't be possible to get here
  }
}

func (h *PlayersHandler) head(game *Game, w http.ResponseWriter,
                              r *http.Request) {}
func (h *PlayersHandler) options(game *Game, w http.ResponseWriter,
                                 r *http.Request) {}

func (h *PlayersHandler) get(game *Game, w http.ResponseWriter,
                             r *http.Request) {
  b, err := json.Marshal(game.GetAllPlayers)
  if (err != nil) {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
  }
  w.Header().Set("Content-Type", "application/json; charset=utf-8")
  fmt.Fprintln(w, string(b))
}

func (h *PlayersHandler) post(game *Game, w http.ResponseWriter,
                              r *http.Request) {
  err := r.ParseForm()
  if err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
  }

  id := r.PostFormValue("id")
  if len(id) == 0 {
    http.Error(w, "Missing arg(s): expecting `id`.", http.StatusBadRequest)
    return
  }

  // Try to add player
  newPlayer, err := game.AddPlayer(id)
  if err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
  }

  // Respond to the request
  newPlayerJson, err := json.Marshal(newPlayer)
  if err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
  }
  http.SetCookie(w, newPlayer.GetCookie())
  fmt.Fprintln(w, string(newPlayerJson))

  return
}
