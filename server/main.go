package main

import (
  "encoding/json"
  "fmt"
  "html/template"
  "log"
  "net/http"
  "time"
  "superghost"
  "sync"
)

var _listeners []chan string
var _listenersMutex sync.RWMutex

var _gs *superghost.GameState

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
  if _, ok := _gs.GetValidCookie(r.Cookies()); ok {
    http.Redirect(w, r, "/", http.StatusFound)
    return
  }
  switch r.Method {

    case http.MethodGet:
      // they've already joined -- redirect them back to the game
      t, _ := template.ParseFiles("join.tmpl")
      t.Execute(w, nil)

    case http.MethodPost:
      r.ParseForm()
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
      r.ParseForm()
      err := _gs.AffixWord(r.Cookies(), r.FormValue("prefix"),
                           r.FormValue("suffix"))
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      fmt.Fprint(w, "success")
      broadcastGameState()

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func challengeIsWordHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {

    case http.MethodPost:
      err := _gs.ChallengeIsWord(r.Cookies())
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

      err := _gs.ChallengeContinuation(r.Cookies())
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
      r.ParseForm()
      err := _gs.RebutChallenge(r.Cookies(), r.FormValue("continuation"))
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

func heartbeatHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {

    case http.MethodPost:
      err := _gs.Heartbeat(r.Cookies())
      if err != nil {
        // they got kicked from the game & came back most likely
        http.Error(w, "player doesn't exist", http.StatusBadRequest)
        return
      }
      fmt.Fprint(w, "success")

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func broadcastGameState() {
  _listenersMutex.Lock()
  defer _listenersMutex.Unlock()

  b, err := json.Marshal(_gs)
  if err != nil {
    panic("couldn't encode superghost") // something's gone terribly wrong
  }
  s := string(b)
  fmt.Println(s) // print all updates to game state to the console!
  for _, c := range _listeners {
    c <- s
  }
  _listeners = make([]chan string, 0) // clear
}

func intermittentlyRemoveDeadPlayers() {
  for _ = range time.Tick(time.Second) {
    go func () {
      if _gs.RemoveDeadPlayers(15 * time.Second) {
        broadcastGameState()
      }
    }()
  }
}

func init() {
  _gs = superghost.NewGameState()
  _listeners = make([]chan string, 0)
  go intermittentlyRemoveDeadPlayers()
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
  http.HandleFunc("/heartbeat", heartbeatHandler)


  err := http.ListenAndServe(":9090", nil) // setting listening port
  if err != nil {
    log.Fatal("ListenAndServe: ", err)
  }
}

