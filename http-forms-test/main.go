package main

import (
  "fmt"
  "html/template"
  "github.com/gorilla/mux"
  "log"
  "net/http"
  // "net/http/httputil"
  // "strings"
)

var word string
messages := make(chan string)

func rootHandler(w http.ResponseWriter, r *http.Request) {
  fmt.Println("method:", r.Method) //get request method
  if r.Method == "GET" {
    t, _ := template.ParseFiles("form.tmpl")
    t.Execute(w, nil)
  } else {
    r.ParseForm()
    // logic part of log in
    word = r.FormValue("prefix") + word + r.FormValue("suffix")
    fmt.Printf("new word: '%s'\n", word)
    fmt.Fprint(w, word)
  }
}

func longPollHandler(w http.ResponseWriter, r *http.Request) {
  if r.Method == "GET" {
    t, _ := template.ParseFiles("form.tmpl")
    t.Execute(w, nil)
  } else {
  }
}

func main() {
  http.HandleFunc("/", rootHandler) // setting router rule
  http.HandleFunc("/long-poll", longPollHandler) // setting router rule

  err := http.ListenAndServe(":9090", nil) // setting listening port
  if err != nil {
    log.Fatal("ListenAndServe: ", err)
  }
}

