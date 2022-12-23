package sgserver

import (
  "encoding/json"
  "fmt"
  "github.com/go-chi/chi/v5"
  "net/http"
  "strconv"
  "strings"
  "superghost"
  "text/template"
  "time"
)

type SuperghostServer struct {
  Rooms map[string]*RoomWrapper
  Router chi.Router
}

func NewSuperghostServer(rooms map[string]*RoomWrapper) *SuperghostServer {
  server := new(SuperghostServer)
  server.Rooms = rooms

  server.Router = chi.NewRouter()
  server.Router.Get("/", server.home)
  server.Router.Get("/rooms", server.rooms)
  server.Router.Post("/rooms", server.rooms)
  server.Router.Get("/rooms/{roomID}", server.room)
  server.Router.Head("/rooms/{roomID}", server.room)
  server.Router.Get("/rooms/{roomID}/join", server.join)
  server.Router.Post("/rooms/{roomID}/join", server.join)
  server.Router.Get("/rooms/{roomID}/next-state", server.nextState)
  server.Router.Get("/rooms/{roomID}/current-state", server.currentState)
  server.Router.Post("/rooms/{roomID}/affix", server.affix)
  server.Router.Post("/rooms/{roomID}/challenge-is-word",
                     server.challengeIsWord)
  server.Router.Post("/rooms/{roomID}/challenge-continuation",
                     server.challengeContinuation)
  server.Router.Post("/rooms/{roomID}/rebuttal", server.rebuttal)
  server.Router.Post("/rooms/{roomID}/concession", server.concession)
  server.Router.Post("/rooms/{roomID}/leave", server.leave)
  server.Router.Post("/rooms/{roomID}/players/{playerID}/votekick",
                     server.votekick)
  server.Router.Get("/rooms/{roomID}/config", server.config)
  server.Router.Post("/rooms/{roomID}/chat", server.chat)
  server.Router.Post("/rooms/{roomID}/ready-up", server.readyUp)
  server.Router.Get("/rooms/{roomID}/next-chat", server.chat)
  server.Router.Post("/rooms/{roomID}/cancellable-leave",
                     server.cancellableLeave)
  server.Router.Post("/rooms/{roomID}/cancel-leave", server.cancelLeave)

  go server.periodicallyDeleteIdleRooms(time.Minute * 10)

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
      t, err := template.ParseFiles("../client/index2.html",
                                    "../client/client_utils.js")
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

    // Send a list of the public games in play
    case http.MethodGet:
      arr := make([]superghost.JRoomMetadata, len(s.Rooms))

      i := 0
      for k := range s.Rooms {
        if !s.Rooms[k].Room.IsPublic() {
          continue
        }
        arr[i] = s.Rooms[k].Room.Metadata(k)
        i++
      }
      b, err := json.Marshal(arr[:i])
      if err != nil {
        http.Error(w, "unexpected internal error",
                   http.StatusInternalServerError)
        panic(err.Error())
      }
      fmt.Fprintln(w, string(b))
      return

    case http.MethodPost:
      // validate params
      err := r.ParseForm()
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      maxPlayers, err := strconv.Atoi(r.FormValue("MaxPlayers"))
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      minWordLength, err := strconv.Atoi(r.FormValue("MinWordLength"))
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      eliminationThreshold, err :=
          strconv.Atoi(r.FormValue("EliminationThreshold"))
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      isPublic := r.FormValue("IsPublic") == "on"
      allowRepeatWords := r.FormValue("AllowRepeatWords") == "on"

      playerTimePerWord, err := strconv.Atoi(r.FormValue("PlayerTimePerWord"))
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
      }

      roomID := superghost.GetRandBase32String(6)
      s.Rooms[roomID] = NewRoomWrapper(superghost.Config{
            MaxPlayers: maxPlayers,
            MinWordLength: minWordLength,
            IsPublic: isPublic,
            EliminationThreshold: eliminationThreshold,
            AllowRepeatWords: allowRepeatWords,
            PlayerTimePerWord: time.Duration(playerTimePerWord) * time.Second,
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
    redirectURIList(w, []string{"/rooms/%s" + roomID})
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
      err := roomWrapper.Room.RebutChallenge(
          r.Cookies(), r.FormValue("prefix"), r.FormValue("suffix"))
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
      b, err := roomWrapper.Room.MarshalJSONFullLog()
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
      myChan := roomWrapper.UpdateListeners.AddListener()
      fmt.Fprint(w, <-myChan)

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (s *SuperghostServer) concession(w http.ResponseWriter, r *http.Request) {
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

func (s *SuperghostServer) cancellableLeave(w http.ResponseWriter,
                                            r *http.Request) {
  roomID := chi.URLParam(r, "roomID")
  roomWrapper, ok := s.Rooms[roomID]
  if !ok {
    http.NotFound(w, r)
    return
  }
  switch r.Method {

    case http.MethodPost:
      err := roomWrapper.Room.ScheduleLeave(r.Cookies())
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      fmt.Fprintln(w, "you are now scheduled to leave the game")

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (s *SuperghostServer) cancelLeave(w http.ResponseWriter, r *http.Request) {
  roomID := chi.URLParam(r, "roomID")
  roomWrapper, ok := s.Rooms[roomID]
  if !ok {
    http.NotFound(w, r)
    return
  }
  switch r.Method {

    case http.MethodPost:
      err := roomWrapper.Room.CancelLeaveIfScheduled(r.Cookies())
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      fmt.Fprintln(w, "you are no longer scheduled to leave the game")

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}


func (s *SuperghostServer) config(w http.ResponseWriter, r *http.Request) {
  roomID := chi.URLParam(r, "roomID")
  roomWrapper, ok := s.Rooms[roomID]
  if !ok {
    http.NotFound(w, r)
    return
  }
  switch r.Method {

    case http.MethodGet:
      b, err := roomWrapper.Room.MarshalJSONConfig()
      if err != nil {
        panic ("couldn't marshal config")
      }
      fmt.Fprint(w, string(b))

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (s *SuperghostServer) chat(w http.ResponseWriter, r *http.Request) {
  roomID := chi.URLParam(r, "roomID")
  roomWrapper, ok := s.Rooms[roomID]
  if !ok {
    http.NotFound(w, r)
    return
  }
  switch r.Method {

    case http.MethodGet:
      myChan := roomWrapper.ChatListeners.AddListener()
      fmt.Fprint(w, <-myChan)


    case http.MethodPost:
      msg, err := roomWrapper.Room.Chat(r.Cookies(), r.FormValue("content"))
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      fmt.Fprint(w, "success")
      // Broadcast chat message to everyone
      b, err := json.Marshal(msg)
      if err != nil {
        panic(err)
      }
      roomWrapper.ChatListeners.Broadcast(string(b))

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (s *SuperghostServer) votekick(w http.ResponseWriter, r *http.Request) {
  roomID := chi.URLParam(r, "roomID")
  roomWrapper, ok := s.Rooms[roomID]
  if !ok {
    http.NotFound(w, r)
    return
  }
  playerID := chi.URLParam(r, "playerID")
  switch r.Method {

    case http.MethodPost:
      err := roomWrapper.Room.Votekick(r.Cookies(), playerID)
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      roomWrapper.BroadcastGameState()

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (s *SuperghostServer) readyUp(w http.ResponseWriter, r *http.Request) {
  roomID := chi.URLParam(r, "roomID")
  roomWrapper, ok := s.Rooms[roomID]
  if !ok {
    http.NotFound(w, r)
    return
  }
  switch r.Method {

    case http.MethodPost:
      err := roomWrapper.Room.ReadyUp(r.Cookies())
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      roomWrapper.BroadcastGameState()

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (s *SuperghostServer) periodicallyDeleteIdleRooms(period time.Duration) {
  ticker := time.NewTicker(period)
  defer ticker.Stop()

  for {
    <-ticker.C
    for key, rw := range s.Rooms {
      if time.Since(rw.Room.LastTouch()) > period {
        delete(s.Rooms, key)
      }
    }
  }
}
