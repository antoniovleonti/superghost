package main

import (
  "fmt"
  "html/template"
  "log"
  "net/http"
  "sync"
  // "net/http/httputil"
  // "strings"
)

var word string
var wordMutex sync.RWMutex

var longPollChannels []chan string
var longPollChannelsMutex sync.RWMutex

func rootHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
    case http.MethodGet:
      t, _ := template.ParseFiles("form.tmpl")
      t.Execute(w, nil)
    case http.MethodPost:
      r.ParseForm()

      wordMutex.Lock() // -Begin critical section-------------------------------
      longPollChannelsMutex.Lock()

      word = r.FormValue("prefix") + word + r.FormValue("suffix")
      fmt.Printf("new word: '%s'\n", word)
      fmt.Fprint(w, word)
      // broadcast new word to all long poll listeners
      for _, c := range longPollChannels {
        c <- word
      }
      longPollChannels = make([]chan string, 0) // clear

      wordMutex.Unlock() // -End critical section-------------------------------
      longPollChannelsMutex.Unlock()
    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func wordHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
    case http.MethodGet:
      wordMutex.RLock()

      fmt.Fprint(w, word)

      wordMutex.RUnlock()
    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func nextWordHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
    case http.MethodGet:
      longPollChannelsMutex.Lock()

      myChan := make(chan string)
      longPollChannels = append(longPollChannels, myChan)

      longPollChannelsMutex.Unlock()

      fmt.Fprint(w, <-myChan)
    default:
      http.Error(w, "", http.StatusMethodNotAllowed)
  }
}

func main() {
  longPollChannels = make([]chan string, 0)

  http.HandleFunc("/", rootHandler) // setting router rule
  http.HandleFunc("/word", wordHandler)
  http.HandleFunc("/long-poll", nextWordHandler)

  err := http.ListenAndServe(":9090", nil) // setting listening port
  if err != nil {
    log.Fatal("ListenAndServe: ", err)
  }
}

