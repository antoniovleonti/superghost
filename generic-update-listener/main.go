package main

import (
  "encoding/json"
  "fmt"
  "html/template"
  "log"
  "net/http"
  "superghost/gamestate"
  "sync"
  "regexp"
)

var _listeners []chan string
var _listenersMutex sync.RWMutex

var _gs *gamestate.GameState

func rootHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
    case http.MethodGet:
      username, ok := _gs.GetValidCookie(r.Cookies())
      if !ok {
        http.Redirect(w, r, "/join", http.StatusFound)
        return
      }
      t, _ := template.ParseFiles("play.tmpl")
      t.Execute(w, struct{Username string}{Username: username})
      return

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func joinHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
    case http.MethodGet:
      t, _ := template.ParseFiles("join.tmpl")
      t.Execute(w, nil)

    case http.MethodPost:
      r.ParseForm()
      if len(r.FormValue("username")) == 0 {
        http.Error(w, "no username provided", http.StatusBadRequest)
        return
      }
      match, err := regexp.MatchString(`^[[:alnum:]]+$`,
                                       r.FormValue("username"))
      if err != nil {
        http.Error(w, "unexpected error while parsing username",
                   http.StatusBadRequest)
        return
      }
      if !match {
        http.Error(w, "username must be alphanumeric", http.StatusBadRequest)
        return
      }
      fmt.Println(r.FormValue("username"))
      cookie, err := _gs.AddPlayer(r.FormValue("username"))
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      http.SetCookie(w, cookie)
      http.Redirect(w, r, "/", http.StatusSeeOther)
      broadcastGameState()
      return

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func wordHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {

    case http.MethodPost:
      if _, ok := _gs.GetInTurnCookie(r.Cookies()); !ok {
        http.Error(w, "request out of turn", http.StatusBadRequest)
        return
      }
      r.ParseForm()

      word, err := _gs.AffixWord(r.FormValue("prefix"), r.FormValue("suffix"))
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }

      fmt.Printf("new word: '%s'\n", word)
      fmt.Fprint(w, "success")
      broadcastGameState()

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func challengeIsWordHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
    case http.MethodPost:
      // it must be your turn to challenge.
      if _, ok := _gs.GetInTurnCookie(r.Cookies()); !ok {
        http.Error(w, "request out of turn", http.StatusBadRequest)
        return
      }

      err := _gs.ChallengeIsWord()
      fmt.Println("I MADE IT")
      if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
      }
      broadcastGameState()

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func challengeContinuationHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
    case http.MethodPost:
      // it must be your turn to challenge.
      if _, ok := _gs.GetInTurnCookie(r.Cookies()); !ok {
        http.Error(w, "request out of turn", http.StatusBadRequest)
        return
      }

      err := _gs.ChallengeContinuation()
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
      }
      broadcastGameState()

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func rebuttalHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
    case http.MethodPost:
      // it must be your turn to challenge.
      if _, ok := _gs.GetInTurnCookie(r.Cookies()); !ok {
        http.Error(w, "request out of turn", http.StatusBadRequest)
        return
      }
      r.ParseForm()
      continuation := r.FormValue("continuation")
      if len(continuation) == 0 {
        http.Error(w, "no continuation provided", http.StatusBadRequest)
        return
      }
      err := _gs.RebutChallenge(continuation)
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      broadcastGameState()

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func stateHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
    case http.MethodGet:
      b, err := json.Marshal(_gs)
      if err != nil {
        panic ("couldn't marshal game state")
      }
      fmt.Fprint(w, string(b))
    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func nextStateHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
    case http.MethodGet:
      _listenersMutex.Lock()
      myChan := make(chan string)
      _listeners = append(_listeners, myChan)
      _listenersMutex.Unlock()

      fmt.Fprint(w, <-myChan)
    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func broadcastGameState() {
  _listenersMutex.Lock()
  defer _listenersMutex.Unlock()

  b, err := json.Marshal(_gs)
  if err != nil {
    panic("couldn't encode gamestate") // something's gone terribly wrong
  }
  s := string(b) // print all updates to game state to the console!
  fmt.Println(s)
  for _, c := range _listeners {
    c <- s
  }
  _listeners = make([]chan string, 0) // clear
}

func init() {
  _gs = gamestate.NewGameState()
  _listeners = make([]chan string, 0)
}

func main() {
  http.HandleFunc("/", rootHandler) // setting router rule
  http.HandleFunc("/join", joinHandler) // setting router rule
  http.HandleFunc("/next-state", nextStateHandler)
  http.HandleFunc("/state", stateHandler)
  http.HandleFunc("/word", wordHandler)
  http.HandleFunc("/challenge-is-word", challengeIsWordHandler)
  http.HandleFunc("/challenge-continuation", challengeContinuationHandler)
  http.HandleFunc("/rebuttal", rebuttalHandler)


  err := http.ListenAndServe(":9090", nil) // setting listening port
  if err != nil {
    log.Fatal("ListenAndServe: ", err)
  }
}

