package main

import (
  "fmt"
  "html/template"
  "log"
  "net/http"
  // "net/http/httputil"
  // "strings"
)

var word string
var longPollChannels []chan string

func rootHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
    case http.MethodGet:
      t, _ := template.ParseFiles("form.tmpl")
      t.Execute(w, nil)
    case http.MethodPost:
      r.ParseForm()
      // logic part of log in
      word = r.FormValue("prefix") + word + r.FormValue("suffix")
      fmt.Printf("new word: '%s'\n", word)
      fmt.Fprint(w, word)
  }
}

func longPollHandler(w http.ResponseWriter, r *http.Request) {
  myChan := make(chan string)
  longPollChannels = append(longPollChannels, myChan)
  fmt.Fprint(w, <-myChan)
}

func main() {
  http.HandleFunc("/", rootHandler) // setting router rule
  http.HandleFunc("/long-poll", longPollHandler) // setting router rule

  err := http.ListenAndServe(":9090", nil) // setting listening port
  if err != nil {
    log.Fatal("ListenAndServe: ", err)
  }
}

