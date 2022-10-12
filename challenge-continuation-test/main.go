package main

import (
  "fmt"
  "html/template"
  "log"
  "net/http"
  "strings"
  // "io"
  // "encoding/json"
  // "net/http/httputil"
  // "strings"
)

type Player struct {
  Id string `json:"id"`
  Cookie http.Cookie `json:"-"`
}

var word string
var inRebutMode bool
var challengeListeners []chan bool
var rebuttalListeners []chan string

func rootHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
    case http.MethodGet:
      t, _ := template.ParseFiles("form.tmpl")
      t.Execute(w, nil)

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func wordHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
    case http.MethodGet:
      fmt.Fprint(w, word)

    case http.MethodPut:
      if inRebutMode {
        http.Error(w, "cannot change word until challenge is addressed",
                   http.StatusBadRequest)
        return
      }
      r.ParseForm()
      word = r.FormValue("newWord")
      fmt.Printf("new word: '%s'\n", word)
      fmt.Fprint(w, word)

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func challengesHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
    case http.MethodGet:
      ch := make(chan bool, 1)
      challengeListeners = append(challengeListeners, ch)
      fmt.Fprint(w, <-ch)

    case http.MethodPost:
      if inRebutMode {
        http.Error(w, "another challenge has already been issued.",
                   http.StatusBadRequest)
        return
      }
      inRebutMode = true
      for _, ch := range challengeListeners {
        ch <- true
      }
      challengeListeners = make([]chan bool, 0)
      fmt.Fprint(w, "challenge issued")

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func rebuttalsHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
    case http.MethodGet:
      ch := make(chan string, 1)
      rebuttalListeners = append(rebuttalListeners, ch)
      fmt.Fprint(w, <-ch)

    case http.MethodPost:
      if !inRebutMode {
        http.Error(w, "cannot rebut unless a challenge has been issued",
                   http.StatusBadRequest)
      }
      r.ParseForm()

      continuation := r.FormValue("continuation")
      if !strings.Contains(continuation, word) {
        http.Error(w, "continuation must contain current substring",
                   http.StatusBadRequest)
        return
      }
      // Response to rebuttal just indicates they have submitted a valid
      // rebuttal i.e. the rebuttal contains the substring
      fmt.Fprint(w, "rebuttal recieved")

      // Now respond to everyone waiting on the result of the challenge
      fmt.Printf("proposed continuation: '%s'\n", continuation)
      reqUri := "https://api.dictionaryapi.dev/api/v2/entries/en/" +
          continuation
      resp, err := http.Get(reqUri)
      if err != nil {
        // handle error
        http.Error(w, "dependency error", http.StatusInternalServerError)
        return
      }
      var result string
      if resp.StatusCode == http.StatusOK {
        result = fmt.Sprintf("'%s' is valid; challenge rebutted.", continuation)
      } else {
        result = fmt.Sprintf("'%s' is invalid; challenge successful.",
                             continuation)
      }
      for _, ch := range rebuttalListeners {
        ch <- result
      }
      rebuttalListeners = make([]chan string, 0)

      inRebutMode = false

    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func main() {
  challengeListeners = make([]chan bool, 0)
  rebuttalListeners = make([]chan string, 0)
  inRebutMode = false

  http.HandleFunc("/", rootHandler) // setting router rule
  http.HandleFunc("/word", wordHandler)
  http.HandleFunc("/challenges", challengesHandler)
  http.HandleFunc("/rebuttals", rebuttalsHandler)

  err := http.ListenAndServe(":9090", nil) // setting listening port
  if err != nil {
    log.Fatal("ListenAndServe: ", err)
  }
}

