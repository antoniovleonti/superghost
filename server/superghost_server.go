package sgserver

import (
  "context"
  "encoding/json"
  "fmt"
  "github.com/go-chi/chi/v5"
  "io"
  "net/http"
  "os"
  "path/filepath"
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
  server.Router.Get("/static/*", server.static)

  server.Router.Route("/rooms", func (r chi.Router) {
    r.Get("/", server.rooms)
    r.Post("/", server.rooms)

    r.Route("/{roomID}", func(r chi.Router) {
      // Middleware to verify roomID is valid and add it to request ctx
      r.Use(server.middlewareGetRoom)

      r.Get("/", server.room)
      r.Head("/", server.room)
      r.Post("/join", server.join)
      r.Get("/next-state", server.nextState)
      r.Get("/current-state", server.currentState)
      r.Post("/affix", server.affix)
      r.Post("/challenge-is-word", server.challengeIsWord)
      r.Post("/challenge-continuation", server.challengeContinuation)
      r.Post("/rebuttal", server.rebuttal)
      r.Post("/concession", server.concession)
      r.Post("/kick", server.kick)
      r.Get("/config", server.config)
      r.Post("/chat", server.chat)
      r.Get("/next-chat", server.chat)
      r.Post("/leave", server.leave)
      r.Post("/cancellable-leave", server.cancellableLeave)
      r.Post("/cancel-leave", server.cancelLeave)
    })
  })

  go server.periodicallyDeleteIdleRooms(time.Minute * 30)

  return server
}

func redirectURIList(w http.ResponseWriter, URI string) {
	w.Header().Set("Content-Type", "text/uri-list; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Location", URI)
	w.WriteHeader(http.StatusFound)
  fmt.Fprint(w, "") // No body, just the header
}

func (s *SuperghostServer) middlewareGetRoom(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ID := chi.URLParam(r, "roomID")
		wrapper, ok := s.Rooms[ID]
		if !ok {
			http.NotFound(w, r)
			return
		}
    ctx := context.WithValue(r.Context(), "roomID", ID)
    ctx = context.WithValue(ctx, "roomWrapper", wrapper)

    next.ServeHTTP(w, r.WithContext(ctx))
  })
}

func (s *SuperghostServer) static(w http.ResponseWriter, r *http.Request) {
  // build the file path by joining the base directory with the requested file
  // path
  filePath := filepath.Join(
      "..", "client", "play", strings.TrimPrefix(r.URL.Path, "/static/"));

  // check if the file exists and is safe to serve
  if fileInfo, err := os.Stat(filePath); err != nil || fileInfo.IsDir() {
    http.NotFound(w, r)
    return
  }

  http.ServeFile(w, r, filePath)
  return
}

func (s *SuperghostServer) home(w http.ResponseWriter, r *http.Request) {
  switch r.Method {

    case http.MethodGet:
      file, err := os.Open("../client/index.html")
      if err != nil {
        http.Error(w, "Internal error", http.StatusInternalServerError)
        panic("../client/index.html couldn't be found!")
      }
      defer file.Close()

      io.Copy(w, file)
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
      pauseAtRoundStart := r.FormValue("PauseAtRoundStart") == "on"

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
            PauseAtRoundStart: pauseAtRoundStart,
          })
      redirectURIList(w, "/rooms/" + roomID)
      return

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (s *SuperghostServer) room(w http.ResponseWriter, r *http.Request) {
  switch r.Method {

    case http.MethodGet:
      file, err := os.Open("../client/play/play.html")
      if err != nil {
        http.Error(w, "Internal error", http.StatusInternalServerError)
        panic("../client/play/play.html couldn't be found!")
      }
      defer file.Close()

      io.Copy(w, file)
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
  ctx := r.Context()
  roomID := ctx.Value("roomID").(string)
  roomWrapper := ctx.Value("roomWrapper").(*RoomWrapper)

  if _, ok := roomWrapper.Room.GetValidCookie(r.Cookies()); ok {
    // they've already joined -- redirect them back to the room
    redirectURIList(w, "/rooms/" + roomID)
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
      fmt.Println("here!")
      r.ParseForm()
      cookie, err := roomWrapper.Room.AddPlayer(r.FormValue("username"),
                                                "/rooms/" + roomID)
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      http.SetCookie(w, cookie)
			fmt.Fprint(w, "")

      roomWrapper.BroadcastGameState()
      return

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (s *SuperghostServer) affix(w http.ResponseWriter, r *http.Request) {
  ctx := r.Context()
  roomWrapper := ctx.Value("roomWrapper").(*RoomWrapper)

  switch r.Method {

    case http.MethodPost:
      r.ParseForm()
      err := roomWrapper.Room.AffixLetter(r.Cookies(), r.FormValue("prefix"),
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
  ctx := r.Context()
  roomWrapper := ctx.Value("roomWrapper").(*RoomWrapper)

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
  ctx := r.Context()
  roomWrapper := ctx.Value("roomWrapper").(*RoomWrapper)

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
  ctx := r.Context()
  roomWrapper := ctx.Value("roomWrapper").(*RoomWrapper)

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
  ctx := r.Context()
  roomWrapper := ctx.Value("roomWrapper").(*RoomWrapper)

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
  ctx := r.Context()
  roomWrapper := ctx.Value("roomWrapper").(*RoomWrapper)

  switch r.Method {

    case http.MethodGet:
      myChan := roomWrapper.UpdateListeners.AddListener()
      fmt.Fprint(w, <-myChan)

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (s *SuperghostServer) concession(w http.ResponseWriter, r *http.Request) {
  ctx := r.Context()
  roomWrapper := ctx.Value("roomWrapper").(*RoomWrapper)

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
  ctx := r.Context()
  roomWrapper := ctx.Value("roomWrapper").(*RoomWrapper)

  switch r.Method {

    case http.MethodPost:
      err := roomWrapper.Room.Leave(r.Cookies())
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      redirectURIList(w, "/")
      roomWrapper.BroadcastGameState()

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func (s *SuperghostServer) cancellableLeave(w http.ResponseWriter,
                                            r *http.Request) {
  ctx := r.Context()
  roomWrapper := ctx.Value("roomWrapper").(*RoomWrapper)

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
  ctx := r.Context()
  roomWrapper := ctx.Value("roomWrapper").(*RoomWrapper)

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
  ctx := r.Context()
  roomWrapper := ctx.Value("roomWrapper").(*RoomWrapper)

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
  ctx := r.Context()
  roomWrapper := ctx.Value("roomWrapper").(*RoomWrapper)

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

func (s *SuperghostServer) kick(w http.ResponseWriter, r *http.Request) {
  ctx := r.Context()
  roomWrapper := ctx.Value("roomWrapper").(*RoomWrapper)

  err := r.ParseForm()
  if err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
  }
  recipient := r.FormValue("Username")

  switch r.Method {

    case http.MethodPost:
      err := roomWrapper.Room.Kick(r.Cookies(), recipient)
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
        s.Rooms[key].Room.Teardown()
        delete(s.Rooms, key)
      }
    }
  }
}
