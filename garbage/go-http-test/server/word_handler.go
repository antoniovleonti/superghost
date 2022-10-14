package main

import (
  "fmt"
  "github.com/gorilla/mux"
  "net/http"
)

type WordHandler struct {
  Manager GamesManager
}

func (h *WordHandler) Handle(w http.ResponseWriter, r *http.Request) {
  gameStr := mux.Vars(r)["game"]
  game, ok := h.Manager.GetGame(gameStr);
  // first check to make sure game is valid
  if !ok {
    http.NotFound(w, r)
    return
  }

  // Validate game
  switch (r.Method) {
    case http.MethodHead:
      h.head(w, r)
    case http.MethodOptions:
      h.options(w, r)
    case http.MethodGet:
      h.get(game, w, r)
    case http.MethodPost:
      h.post(game, w, r)
    default:
      // Shouldn't be possible to get here
  }
}

func (h *WordHandler) head(w http.ResponseWriter, r *http.Request) {
}

func (h *WordHandler) options(w http.ResponseWriter, r *http.Request) {
}

func (h *WordHandler) get(game *Game, w http.ResponseWriter, r *http.Request) {
  fmt.Fprintln(w, game.GetWord())
}

func (h *WordHandler) post(game *Game, w http.ResponseWriter, r *http.Request) {
  if !game.RequestContainsInTurnCookie(r) {
    http.Error(w, "Request made out of turn (did you send your cookie?).",
               http.StatusForbidden)
    return
  }

  err := r.ParseForm()
  if err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
  }

  prefix := r.PostFormValue("prefix")
  suffix := r.PostFormValue("suffix")
  // Verify exactly 1 affix is provided
  if (len(prefix) == 0) == (len(suffix) == 0) { // !(A xor B)
    http.Error(w, "Missing arg: expecting one of `prefix` or `suffix`.",
               http.StatusBadRequest)
    return
  }
  // Verify the provided affix is of length 1.
  if (len(prefix) == 1) == (len(suffix) == 1) { // !(A xor B)
    http.Error(w, "Affixes can only be 1 character long.",
               http.StatusBadRequest)
    return
  }

  var updatedWord string
  if len(prefix) > len(suffix) {
    updatedWord = game.PostPrefix(prefix)
  } else {
    updatedWord = game.PostSuffix(suffix)
  }
  fmt.Fprintln(w, updatedWord)
  return
}
