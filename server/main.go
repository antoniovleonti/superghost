package main

import (
  "fmt"
  "github.com/go-chi/chi/v5"
  "net/http"
  "strconv"
  "strings"
  "superghost"
  "sync"
  "text/template"
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
  Router chi.Router
}

func NewSuperghostServer() *SuperghostServer {
  server := new(SuperghostServer)
  server.Rooms = make(map[string]*RoomWrapper)

  server.Router = chi.NewRouter()
  server.Router.Get("/", server.home)
  server.Router.Post("/rooms", server.rooms)
  server.Router.Get("/rooms/{roomID}", server.room)
  server.Router.Head("/rooms/{roomID}", server.room)
  server.Router.Get("/rooms/{roomID}/join", server.join)
  server.Router.Post("/rooms/{roomID}/join", server.join)
  server.Router.Get("/rooms/{roomID}/next-state", server.nextState)
  server.Router.Get("/rooms/{roomID}/current-state", server.currentState)
  server.Router.Post("/rooms/{roomID}/affix", server.affix)
  server.Router.Post("/rooms/{roomID}/challenge-is-word", server.challengeIsWord)
  server.Router.Post(
      "/rooms/{roomID}/challenge-continuation", server.challengeContinuation)
  server.Router.Post("/rooms/{roomID}/rebuttal", server.rebuttal)
  server.Router.Post("/rooms/{roomID}/concession", server.concession)
  server.Router.Post("/rooms/{roomID}/leave", server.leave)

  return server
}

func redirectURIList(w http.ResponseWriter, URIs []string) {
	w.Header().Set("Content-Type", "text/uri-list; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusSeeOther)
	fmt.Fprintln(w, strings.Join(URIs, "\n"))
}

func (s *SuperghostServer) home(w http.ResponseWriter, r *http.Request) {
  switch r.Method {

    case http.MethodGet:
      t, err := template.ParseFiles("../client/index.html")
      if err != nil {
        http.Error(w, "unexpected error", http.StatusInternalServerError)
        panic(err.Error())
      }
      err = t.Execute(w, map[string] string {})
      return

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (s *SuperghostServer) rooms(w http.ResponseWriter, r *http.Request) {
  switch r.Method {

    case http.MethodPost:
      // validate params
      parseFormErr := r.ParseForm()
      if parseFormErr != nil {
        http.Error(w, parseFormErr.Error(), http.StatusBadRequest)
        return
      }
      maxPlayers, maxPlayersErr := strconv.Atoi(r.FormValue("maxPlayers"))
      if maxPlayersErr != nil {
        http.Error(w, maxPlayersErr.Error(), http.StatusBadRequest)
        return
      }
      minWordLength, minWordLengthErr :=
          strconv.Atoi(r.FormValue("minWordLength"))
      if minWordLengthErr != nil {
        http.Error(w, minWordLengthErr.Error(), http.StatusBadRequest)
        return
      }
      isPublic := r.FormValue("isPublic") == "on"

      roomID := superghost.GetRandBase32String(6)
      s.Rooms[roomID] = NewRoomWrapper(superghost.Config{
            MaxPlayers: maxPlayers,
            MinWordLength: minWordLength,
            IsPublic: isPublic,
          })
      redirectURIList(w, []string{"/rooms/" + roomID + "/join"})
      return

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (s *SuperghostServer) room(w http.ResponseWriter, r *http.Request) {
  roomID := chi.URLParam(r, "roomID")
  roomWrapper, ok := s.Rooms[roomID]
  if !ok {
    fmt.Println(roomID + " is not a valid room")
    http.NotFound(w, r)
    return
  }
  switch r.Method {

    case http.MethodGet:
      username, ok := roomWrapper.Room.GetValidCookie(r.Cookies())
      if !ok {
        http.Redirect(w, r, fmt.Sprintf("/rooms/%s/join", roomID),
                      http.StatusFound)
        return
      }
      t, err := template.ParseFiles("../client/play2.html",
                                    "../client/client_utils.js")
      if err != nil {
        panic(err.Error())
      }
      err = t.Execute(w, map[string] string {"Username": username,
                                             "RoomID": roomID})
      if err != nil {
        panic(err.Error())
      }
      return

    case http.MethodHead:
      w.WriteHeader(http.StatusOK)
      fmt.Fprint(w, "") // No body, just the header
      return

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (s *SuperghostServer) join(w http.ResponseWriter, r *http.Request) {
  roomID := chi.URLParam(r, "roomID")
  roomWrapper, ok := s.Rooms[roomID]
  if !ok {
    http.NotFound(w, r)
    return
  }

  if _, ok := roomWrapper.Room.GetValidCookie(r.Cookies()); ok {
    // they've already joined -- redirect them back to the room
    http.Redirect(w, r, fmt.Sprintf("/rooms/%s", roomID), http.StatusFound)
    return
  }
  switch r.Method {

    case http.MethodGet:
      t, err := template.ParseFiles("../client/join.html",
                                    "../client/client_utils.js")
      if err != nil {
        panic(err.Error())
      }
      t.Execute(w, map[string] string {"RoomID": roomID})

    case http.MethodPost:
      r.ParseForm()
      cookie, err := roomWrapper.Room.AddPlayer(r.FormValue("username"),
                                                "/rooms/" + roomID)
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      http.SetCookie(w, cookie)
      redirectURIList(w, []string{"/rooms/" + roomID})
      roomWrapper.BroadcastGameState()
      return

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (s *SuperghostServer) affix(w http.ResponseWriter, r *http.Request) {
  roomID := chi.URLParam(r, "roomID")
  roomWrapper, ok := s.Rooms[roomID]
  if !ok {
    http.NotFound(w, r)
    return
  }
  switch r.Method {

    case http.MethodPost:
      r.ParseForm()
      err := roomWrapper.Room.AffixWord(r.Cookies(), r.FormValue("prefix"),
                            r.FormValue("suffix"))
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      fmt.Fprint(w, "success")
      roomWrapper.BroadcastGameState()

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (s *SuperghostServer) challengeIsWord(w http.ResponseWriter,
                                             r *http.Request) {
  roomID := chi.URLParam(r, "roomID")
  roomWrapper, ok := s.Rooms[roomID]
  if !ok {
    http.NotFound(w, r)
    return
  }
  switch r.Method {

    case http.MethodPost:
      err := roomWrapper.Room.ChallengeIsWord(r.Cookies())
      if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
      }
      roomWrapper.BroadcastGameState()

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (s *SuperghostServer) challengeContinuation(w http.ResponseWriter,
                                                   r *http.Request) {
  roomID := chi.URLParam(r, "roomID")
  roomWrapper, ok := s.Rooms[roomID]
  if !ok {
    http.NotFound(w, r)
    return
  }
  switch r.Method {
    case http.MethodPost:

      err := roomWrapper.Room.ChallengeContinuation(r.Cookies())
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      roomWrapper.BroadcastGameState()

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (s *SuperghostServer) rebuttal(w http.ResponseWriter, r *http.Request) {
  roomID := chi.URLParam(r, "roomID")
  roomWrapper, ok := s.Rooms[roomID]
  if !ok {
    http.NotFound(w, r)
    return
  }
  switch r.Method {
    case http.MethodPost:
      // it must be your turn to challenge.
      r.ParseForm()
      giveUp, err := strconv.ParseBool(r.FormValue("giveUp"))
      if err != nil {
        giveUp = false
      }
      err = roomWrapper.Room.RebutChallenge(
          r.Cookies(), r.FormValue("continuation"), giveUp)
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      roomWrapper.BroadcastGameState()

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (s *SuperghostServer) currentState(w http.ResponseWriter,
                                          r *http.Request) {
  roomID := chi.URLParam(r, "roomID")
  roomWrapper, ok := s.Rooms[roomID]
  if !ok {
    http.NotFound(w, r)
    return
  }
  switch r.Method {
    case http.MethodGet:
      b, err := roomWrapper.Room.GetJsonGameState()
      if err != nil {
        panic ("couldn't marshal room state")
      }
      fmt.Fprint(w, string(b))
    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (s *SuperghostServer) nextState(w http.ResponseWriter, r *http.Request) {
  roomID := chi.URLParam(r, "roomID")
  roomWrapper, ok := s.Rooms[roomID]
  if !ok {
    http.NotFound(w, r)
    return
  }
  switch r.Method {

    case http.MethodGet:
      roomWrapper.ListenersMutex.Lock()
      myChan := make(chan string)
      roomWrapper.Listeners = append(roomWrapper.Listeners, myChan)
      roomWrapper.ListenersMutex.Unlock()

      fmt.Fprint(w, <-myChan)

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (s *SuperghostServer) heartbeat(w http.ResponseWriter, r *http.Request) {
  roomID := chi.URLParam(r, "roomID")
  roomWrapper, ok := s.Rooms[roomID]
  if !ok {
    http.NotFound(w, r)
    return
  }
  switch r.Method {

    case http.MethodPost:
      err := roomWrapper.Room.Heartbeat(r.Cookies())
      if err != nil {
        // they got kicked from the room & came back most likely
        http.Error(w, "couldn't find player", http.StatusNotFound)
        return
      }
      fmt.Fprint(w, "success")

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (s *SuperghostServer) concession(w http.ResponseWriter,
                                        r *http.Request) {
  roomID := chi.URLParam(r, "roomID")
  roomWrapper, ok := s.Rooms[roomID]
  if !ok {
    http.NotFound(w, r)
    return
  }
  switch r.Method {

    case http.MethodPost:
      err := roomWrapper.Room.Concede(r.Cookies())
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      fmt.Fprint(w, "success")
      roomWrapper.BroadcastGameState()

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (s *SuperghostServer) leave(w http.ResponseWriter, r *http.Request) {
  roomID := chi.URLParam(r, "roomID")
  roomWrapper, ok := s.Rooms[roomID]
  if !ok {
    http.NotFound(w, r)
    return
  }
  switch r.Method {

    case http.MethodPost:
      err := roomWrapper.Room.Leave(r.Cookies())
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      redirectURIList(w, []string{"/"})
      roomWrapper.BroadcastGameState()

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (rw *RoomWrapper) BroadcastGameState() {
  rw.ListenersMutex.Lock()
  defer rw.ListenersMutex.Unlock()

  b, err := rw.Room.GetJsonGameState()
  if err != nil {
    panic("couldn't get json room state") // something's gone terribly wrong
  }
  s := string(b)
  fmt.Println(s) // print all updates to room state to the console!
  for _, c := range rw.Listeners {
    c <- s
  }
  rw.Listeners = make([]chan string, 0) // clear
}

// func (s *SuperghostServer) intermittentlyRemoveDeadPlayers() {
  //for _ = range time.Tick(time.Second) {
    //go func () {
      //if roomWrapper.Room.RemoveDeadPlayers(10 * time.Minute) {
        //roomWrapper.BroadcastGameState()
      //}
    //}()
  //}
//}

func main() {
  server := NewSuperghostServer()
  server.Rooms["test-room"] = NewRoomWrapper(superghost.Config{
        MaxPlayers: 5,
        MinWordLength: 5,
        IsPublic: true,
      })

  http.ListenAndServe(":9090", server.Router)
}

