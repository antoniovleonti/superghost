package main

import (
  "github.com/gorilla/mux"
  "net/http"
)

type ChallengeIsWordHandler struct {
  Manager GamesManager
}

func (h *ChallengeIsWordHandler) Handle(w http.ResponseWriter,
                                        r *http.Request) {
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
    case http.MethodOptions:
    case http.MethodPost:
      h.post(game, w, r)
    default:
      // Shouldn't be possible to get here
  }
}

func (h *ChallengeIsWordHandler) post(game *Game, w http.ResponseWriter,
                                      r *http.Request) {
  if !game.RequestContainsInTurnCookie(r) {
    http.Error(w, "Request made out of turn (did you send your cookie?).",
               http.StatusForbidden)
    return
  }

  // create http request to test word validity
}
