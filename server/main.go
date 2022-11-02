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
  // "github.com/gorilla/mux"
)

type RoomWrapper struct {
  Listeners []chan string
  ListenersMutex sync.RWMutex
  Room *superghost.Room
}

func NewRoomWrapper(config superghost.Config) *RoomWrapper {
  rh := new(RoomWrapper)
  rh.Room = superghost.NewRoom(config)
  rh.Listeners = make([]chan string, 0)
  return rh
}

type SuperghostServer struct {
  Rooms map[string]*RoomWrapper
}

func NewSuperghostServer() *SuperghostServer {
  server := new(SuperghostServer)
  server.Rooms = make(map[string]*RoomWrapper)
  return server
}

func (sgs *SuperghostServer) rootHandler(w http.ResponseWriter,
                                         r *http.Request) {
  switch r.Method {

    case http.MethodGet:
      username, ok := sgs.Rooms["asdf"].Room.GetValidCookie(r.Cookies())
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

func (sgs *SuperghostServer) joinHandler(w http.ResponseWriter,
                                         r *http.Request) {
  if _, ok := sgs.Rooms["asdf"].Room.GetValidCookie(r.Cookies()); ok {
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
      cookie, err := sgs.Rooms["asdf"].Room.AddPlayer(r.FormValue("username"))
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      http.SetCookie(w, cookie)
      http.Redirect(w, r, "/", http.StatusSeeOther)
      sgs.Rooms["asdf"].BroadcastGameState()
      return

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (sgs *SuperghostServer) wordHandler(w http.ResponseWriter,
                                         r *http.Request) {
  switch r.Method {

    case http.MethodPost:
      r.ParseForm()
      err := sgs.Rooms["asdf"].Room.AffixWord(r.Cookies(), r.FormValue("prefix"),
                            r.FormValue("suffix"))
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      fmt.Fprint(w, "success")
      sgs.Rooms["asdf"].BroadcastGameState()

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (sgs *SuperghostServer) challengeIsWordHandler(w http.ResponseWriter,
                                                    r *http.Request) {
  switch r.Method {

    case http.MethodPost:
      err := sgs.Rooms["asdf"].Room.ChallengeIsWord(r.Cookies())
      if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
      }
      sgs.Rooms["asdf"].BroadcastGameState()

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (sgs *SuperghostServer) challengeContinuationHandler(w http.ResponseWriter, 
                                                          r *http.Request) {
  switch r.Method {
    case http.MethodPost:

      err := sgs.Rooms["asdf"].Room.ChallengeContinuation(r.Cookies())
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      sgs.Rooms["asdf"].BroadcastGameState()

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (sgs *SuperghostServer) rebuttalHandler(w http.ResponseWriter,
                                             r *http.Request) {
  switch r.Method {
    case http.MethodPost:
      // it must be your turn to challenge.
      r.ParseForm()
      giveUp, err := strconv.ParseBool(r.FormValue("giveUp"))
      if err != nil {
        giveUp = false
      }
      err = sgs.Rooms["asdf"].Room.RebutChallenge(r.Cookies(), r.FormValue("continuation"),
                                 giveUp)
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      sgs.Rooms["asdf"].BroadcastGameState()

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (sgs *SuperghostServer) stateHandler(w http.ResponseWriter,
                                          r *http.Request) {
  switch r.Method {
    case http.MethodGet:
      b, err := sgs.Rooms["asdf"].Room.GetJsonGameState()
      if err != nil {
        panic ("couldn't marshal game state")
      }
      fmt.Fprint(w, string(b))
    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (sgs *SuperghostServer) nextStateHandler(w http.ResponseWriter,
                                              r *http.Request) {
  switch r.Method {

    case http.MethodGet:
      sgs.Rooms["asdf"].ListenersMutex.Lock()
      myChan := make(chan string)
      sgs.Rooms["asdf"].Listeners = append(sgs.Rooms["asdf"].Listeners, myChan)
      sgs.Rooms["asdf"].ListenersMutex.Unlock()

      fmt.Fprint(w, <-myChan)

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (sgs *SuperghostServer) heartbeatHandler(w http.ResponseWriter,
                                              r *http.Request) {
  switch r.Method {

    case http.MethodPost:
      err := sgs.Rooms["asdf"].Room.Heartbeat(r.Cookies())
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

func (sgs *SuperghostServer) concessionHandler(w http.ResponseWriter,
                                               r *http.Request) {
  switch r.Method {

    case http.MethodPost:
      err := sgs.Rooms["asdf"].Room.Concede(r.Cookies())
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      fmt.Fprint(w, "success")
      sgs.Rooms["asdf"].BroadcastGameState()

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (sgs *SuperghostServer) leaveHandler(w http.ResponseWriter,
                                          r *http.Request) {
  switch r.Method {

    case http.MethodPost:
      err := sgs.Rooms["asdf"].Room.Leave(r.Cookies())
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      http.Redirect(w, r, "/join", http.StatusFound)
      sgs.Rooms["asdf"].BroadcastGameState()

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (rw *RoomWrapper) BroadcastGameState() {
  rw.ListenersMutex.Lock()
  defer rw.ListenersMutex.Unlock()

  b, err := rw.Room.GetJsonGameState()
  if err != nil {
    panic("couldn't get json game state") // something's gone terribly wrong
  }
  s := string(b)
  fmt.Println(s) // print all updates to game state to the console!
  for _, c := range rw.Listeners {
    c <- s
  }
  rw.Listeners = make([]chan string, 0) // clear
}

func (sgs *SuperghostServer) intermittentlyRemoveDeadPlayers() {
  for _ = range time.Tick(time.Second) {
    go func () {
      if sgs.Rooms["asdf"].Room.RemoveDeadPlayers(10 * time.Minute) {
        sgs.Rooms["asdf"].BroadcastGameState()
      }
    }()
  }
}

func main() {
  sgs := NewSuperghostServer()
  sgs.Rooms["asdf"] = NewRoomWrapper(superghost.Config{
        MaxPlayers: 5,
        MinWordLength: 5,
        IsPublic: true,
      })

  http.HandleFunc("/", sgs.rootHandler) // setting router rule
  http.HandleFunc("/join", sgs.joinHandler) // setting router rule
  http.HandleFunc("/next-state", sgs.nextStateHandler)
  http.HandleFunc("/state", sgs.stateHandler)
  http.HandleFunc("/word", sgs.wordHandler)
  http.HandleFunc("/challenge-is-word", sgs.challengeIsWordHandler)
  http.HandleFunc("/challenge-continuation", sgs.challengeContinuationHandler)
  http.HandleFunc("/rebuttal", sgs.rebuttalHandler)
  http.HandleFunc("/heartbeat", sgs.heartbeatHandler)
  http.HandleFunc("/concession", sgs.concessionHandler)
  http.HandleFunc("/leave", sgs.leaveHandler)

  err := http.ListenAndServe(":9090", nil) // setting listening port
  if err != nil {
    log.Fatal("ListenAndServe: ", err)
  }
}

