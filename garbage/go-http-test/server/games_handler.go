package main

import (
  "encoding/json"
  "fmt"
  "net/http"
)

type GamesHandler struct {
  Manager GamesManager
}

func (h GamesHandler) Handle (w http.ResponseWriter, r *http.Request) {
  // Common operations

  switch (r.Method) {
    case http.MethodHead:
      h.head(w, r)
    case http.MethodOptions:
      h.options(w, r)
    case http.MethodGet:
      h.get(w, r)
    case http.MethodPost:
      h.post(w, r)
    default:
      // Shouldn't be possible to get here
  }
}

func (h *GamesHandler) head(w http.ResponseWriter, r *http.Request) {
}

func (h *GamesHandler) options(w http.ResponseWriter, r *http.Request) {
}

func (h *GamesHandler) get(w http.ResponseWriter, r *http.Request) {
  b, err := json.Marshal(h.Manager.GetAllGames())
  if (err != nil) {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
  }
  w.Header().Set("Content-Type", "application/json; charset=utf-8")
  fmt.Fprintln(w, string(b))
}

func (h *GamesHandler) post(w http.ResponseWriter, r *http.Request) {
  r.ParseForm()

  playerId := r.PostFormValue("playerId")
  if len(playerId) == 0 {
    http.Error(w, "Missing arg: expecting `playerId`.", http.StatusBadRequest)
  }
  newGame, cookie := h.Manager.CreateGame(playerId)

  b, err := json.Marshal(newGame)
  if (err != nil) {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
  }
  http.SetCookie(w, cookie)
  w.Header().Set("Content-Type", "application/json; charset=utf-8")
  fmt.Fprintln(w, string(b))
}
