package sgserver

import (
  "sync"
)

type ListenerGroup struct {
  Listeners []chan string
  ListenersMutex sync.RWMutex
}

func newListenerGroup() *ListenerGroup {
  lg := new(ListenerGroup)
  lg.Listeners = make([]chan string, 0)
  return lg
}

func (lg *ListenerGroup) AddListener() chan string {
  lg.ListenersMutex.Lock()
  defer lg.ListenersMutex.Unlock()

  newChan := make(chan string)
  lg.Listeners = append(lg.Listeners, newChan)
  return newChan
}

func (lg *ListenerGroup) Broadcast(s string) {
  lg.ListenersMutex.Lock()
  defer lg.ListenersMutex.Unlock()

  for _, c := range lg.Listeners {
    c <- s
  }
  lg.Listeners = make([]chan string, 0) // clear
}
