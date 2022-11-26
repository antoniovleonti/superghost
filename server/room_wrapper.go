package sgserver

import (
  "superghost"
)

type RoomWrapper struct {
  UpdateListeners *ListenerGroup
  ChatListeners *ListenerGroup
  Room *superghost.Room
}

func NewRoomWrapper(config superghost.Config) *RoomWrapper {
  rw := new(RoomWrapper)
  rw.Room = superghost.NewRoom(config)
  rw.UpdateListeners = newListenerGroup()
  rw.ChatListeners = newListenerGroup()
  return rw
}

func (rw *RoomWrapper) BroadcastGameState() {
  b, err := rw.Room.MarshalJSON()
  if err != nil {
    panic("couldn't get json room state") // something's gone terribly wrong
  }
  s := string(b)
  rw.UpdateListeners.Broadcast(s)
}

