package main

import (
  "fmt"
  "html/template"
  "log"
  "net/http"
  "sync"
  // "io"
  // "encoding/json"
  "superghost/gamestate"
  // "net/http/httputil"
  // "strings"
)

var updateListeners []chan string
var updateListenersMutex sync.RWMutex

var gs *gamestate.GameState

func rootHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
    case http.MethodGet:
      if !gs.ContainsValidCookie(r.Cookies()) {
        http.Redirect(w, r, "/join", http.StatusFound)
        return
      }
      t, _ := template.ParseFiles("form.tmpl")
      t.Execute(w, nil)
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
      cookie, err := gs.AddPlayer(r.FormValue("username"))
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      http.SetCookie(w, cookie)
      http.Redirect(w, r, "/", http.StatusSeeOther)
      return

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func wordHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
    case http.MethodGet:
      fmt.Fprint(w, gs.GetWord())

    case http.MethodPost:
      r.ParseForm()

      updateListenersMutex.Lock()

      word, err := gs.AffixWord(r.FormValue("prefix"), r.FormValue("suffix"))
      if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
      }
      fmt.Printf("new word: '%s'\n", word)
      fmt.Fprint(w, word)
      // broadcast new word to all long poll listeners
      for _, c := range updateListeners {
        c <- word
      }
      updateListeners = make([]chan string, 0) // clear

      updateListenersMutex.Unlock()
    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func nextWordHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
    case http.MethodGet:
      updateListenersMutex.Lock()

      myChan := make(chan string)
      updateListeners = append(updateListeners, myChan)

      updateListenersMutex.Unlock()

      fmt.Fprint(w, <-myChan)
    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func isWordHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
    case http.MethodGet:
      isValid, err := gs.ValidateWord()
      if err != nil {
        // handle error
        http.Error(w, "Error validating word", http.StatusInternalServerError)
        return
      }
      fmt.Fprint(w, isValid)
      return

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func main() {
  gs = gamestate.NewGameState()
  updateListeners = make([]chan string, 0)

  http.HandleFunc("/", rootHandler) // setting router rule
  http.HandleFunc("/join", joinHandler) // setting router rule
  http.HandleFunc("/word", wordHandler)
  http.HandleFunc("/next-word", nextWordHandler)
  http.HandleFunc("/is-word", isWordHandler)

  err := http.ListenAndServe(":9090", nil) // setting listening port
  if err != nil {
    log.Fatal("ListenAndServe: ", err)
  }
}

