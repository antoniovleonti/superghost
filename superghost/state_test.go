package superghost

import (
  "testing"
  "strconv"
  "fmt"
)

func TestAddPlayer(t *testing.T) {
  gs := newState()

  _, err := gs.addPlayer("player1", 1)
  if err != nil {
    t.Errorf("couldn't add first player to game")
  }
  // should not accept player when at limit
  _, err = gs.addPlayer("player2", 1)
  if err == nil {
    t.Errorf("maxPlayers not respected")
  }
  // should not accept duplicate player
  _, err = gs.addPlayer("player1", 2)
  if err == nil {
    t.Errorf("duplicate player added")
  }
  // should not accept player with illegal name
  _, err = gs.addPlayer(" !@#$%^$&*()", 2)
  if err == nil {
    t.Errorf("player with illegal name added")
  }

  // there should only be one player still
  if len(gs.players) != 1 {
    t.Errorf("|gs.players| = %d (expected 1)", len(gs.players))
  }
}
