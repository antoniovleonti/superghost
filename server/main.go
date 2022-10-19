package main

import (
  "fmt"
  "log"
  "net/http"
  "strconv"
  "superghost"
  "sync"
  "text/template"
  "time"
)

var _listeners []chan string
var _listenersMutex sync.RWMutex
var _sgg *superghost.Room

func rootHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {

    case http.MethodGet:
      username, ok := _sgg.GetValidCookie(r.Cookies())
      if !ok {
        http.Redirect(w, r, "/join", http.StatusFound)
        return
      }
      t, err := template.ParseFiles("../client/play.html",
                                    "../client/script.js",
                                    "../client/style.css",
                                    "../client/sharedHtml.tmpl")
      if err != nil {
        fmt.Println(err.Error())
        panic(err.Error())
      }
      err = t.Execute(w, map[string] string {"Username": username})
      if err != nil {
        fmt.Println(err.Error())
        panic(err.Error())
      }
      return

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func joinHandler(w http.ResponseWriter, r *http.Request) {
  if _, ok := _sgg.GetValidCookie(r.Cookies()); ok {
    // they've already joined -- redirect them back to the game
    http.Redirect(w, r, "/", http.StatusFound)
    return
  }
  switch r.Method {

    case http.MethodGet:
      t, err := template.ParseFiles("../client/join.html",
                                    "../client/style.css",
                                    "../client/sharedHtml.tmpl")
      if err != nil {
        fmt.Println(err.Error())
        panic(err.Error())
      }
      t.Execute(w, map[string] string {"GameId": "/(gameid)"})

    case http.MethodPost:
      r.ParseForm()
      cookie, err := _sgg.AddPlayer(r.FormValue("username"))
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
      err := _sgg.AffixWord(r.Cookies(), r.FormValue("prefix"),
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
      err := _sgg.ChallengeIsWord(r.Cookies())
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

      err := _sgg.ChallengeContinuation(r.Cookies())
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
      giveUp, err := strconv.ParseBool(r.FormValue("giveUp"))
      if err != nil {
        giveUp = false
      }
      err = _sgg.RebutChallenge(r.Cookies(), r.FormValue("continuation"),
                                 giveUp)
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
      b, err := _sgg.GetJsonGameState()
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
      err := _sgg.Heartbeat(r.Cookies())
      if err != nil {
        // they got kicked from the game & came back most likely
        http.Error(w, "couldn't find player", http.StatusNotFound)
        return
      }
      fmt.Fprint(w, "success")

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func concessionHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {

    case http.MethodPost:
      err := _sgg.Concede(r.Cookies())
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
      }
      fmt.Fprint(w, "success")
      broadcastGameState()

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func broadcastGameState() {
  _listenersMutex.Lock()
  defer _listenersMutex.Unlock()

  b, err := _sgg.GetJsonGameState()
  if err != nil {
    panic("couldn't get json game state") // something's gone terribly wrong
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
      if _sgg.RemoveDeadPlayers(15 * time.Second) {
        broadcastGameState()
      }
    }()
  }
}

func init() {
  _sgg = superghost.NewRoom(superghost.Config{
        MaxPlayers: 3,
        MinWordLength: 5,
        IsPublic: true,
      })
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
  http.HandleFunc("/concession", concessionHandler)

  err := http.ListenAndServe(":9090", nil) // setting listening port
  if err != nil {
    log.Fatal("ListenAndServe: ", err)
  }
}

