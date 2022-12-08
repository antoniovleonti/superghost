package sgserver

import (
  "superghost"
  "fmt"
  "time"
)

type RoomWrapper struct {
  Room *superghost.Room

  UpdateListeners *ListenerGroup
  ChatListeners *ListenerGroup

  asyncUpdateCh chan bool
}

func NewRoomWrapper(config superghost.Config) *RoomWrapper {
  rw := new(RoomWrapper)

  rw.asyncUpdateCh = make(chan bool)
  rw.Room = superghost.NewRoom(config, rw.asyncUpdateCh)

  rw.UpdateListeners = newListenerGroup()
  rw.ChatListeners = newListenerGroup()

  go rw.ListenForAsyncUpdateSignals()

  return rw
}

func (rw *RoomWrapper) BroadcastGameState() {
  b, err := rw.Room.MarshalJSON()
  if err != nil {
    panic("couldn't get json room state") // something's gone terribly wrong
  }
  s := string(b)
  // For debugging purposes, print the game state
  fmt.Println(time.Now().String() + ": "  + s)
  rw.UpdateListeners.Broadcast(s)
}

func (rw *RoomWrapper) ListenForAsyncUpdateSignals() {
  for {
    <-rw.asyncUpdateCh
    rw.BroadcastGameState()
  }
}
