package superghost

import (
  "testing"
  "time"
  "net/http"
  "strconv"
)

type testRoomUtils struct {
  room *Room
  asyncUpdateCh chan bool
  usernameToCookie map[string]*http.Cookie
}

func newTestRoomUtils(config Config) *testRoomUtils {
  tru := new(testRoomUtils)
  tru.asyncUpdateCh = make(chan bool)
  tru.room = NewRoom(config, tru.asyncUpdateCh)
  tru.usernameToCookie = make(map[string]*http.Cookie)
  return tru
}

// Adds n players to game named "1", "2", ..
func (tru *testRoomUtils) addNPlayers(n int) error {
  start := len(tru.room.pm.players)
  for i := start; i < start + n; i++ {
    username := strconv.Itoa(i)
    var err error
    tru.usernameToCookie[username], err = tru.room.AddPlayer(username, "xyz")
    if err != nil {
      return err
    }
  }
  return nil
}

func (tru *testRoomUtils) readyUpAllPlayers() error {
  for username, cookie := range tru.usernameToCookie {
    if tru.room.pm.usernameToPlayer[username].isReady {
      continue
    }
    err := tru.room.ReadyUp([]*http.Cookie{cookie})
    if err != nil {
      return err
    }
  }
  return nil
}

func (tru *testRoomUtils) currentPlayerCookies() []*http.Cookie {
  return []*http.Cookie{tru.room.pm.currentPlayer().cookie}
}

func TestReadyUpStartsGame(t *testing.T) {
  tru := newTestRoomUtils(Config {
    MaxPlayers: 16,
    MinWordLength: 5,
    IsPublic: true,
    EliminationThreshold: 0,
    AllowRepeatWords: false,
    PlayerTimePerWord: time.Second * 0,
  })

  err := tru.addNPlayers(2)
  if err != nil {
    t.Errorf("couldn't add two players: " + err.Error())
  }

  err = tru.readyUpAllPlayers()
  if err != nil {
    t.Errorf("couldn't ready up all players: " + err.Error())
  }

  // at this point server will send out the update and users will be notified
  // if the game started
  if tru.room.state != kEdit {
    t.Errorf("didn't enter edit mode")
  }
}

func TestGameEndsWhenOnePlayerRemains(t *testing.T) {
}

func TestVotekickCurrentPlayer(t *testing.T) {
  tru := newTestRoomUtils(Config {
    MaxPlayers: 16,
    MinWordLength: 5,
    IsPublic: true,
    EliminationThreshold: 0,
    AllowRepeatWords: false,
    PlayerTimePerWord: time.Second * 60,
  })

  err := tru.addNPlayers(3)
  if err != nil {
    t.Errorf("couldn't add players: " + err.Error())
  }

  // Edit the first player's time to make sure when they are kicked their
  // remaining time does not spill over to the next person (bug as of time of
  // writing this test). Time starts at 60.
  tru.room.pm.currentPlayer().timeRemaining = 30 * time.Second

  err = tru.readyUpAllPlayers()
  if err != nil {
    t.Errorf("couldn't ready up all players: " + err.Error())
  }

  preKickDeadline := tru.room.pm.currentPlayerDeadline

  err = tru.room.Votekick([]*http.Cookie{tru.usernameToCookie["1"]}, "0")
  if err != nil {
    t.Errorf("couldn't vote to kick player 0: " + err.Error())
  }
  err = tru.room.Votekick([]*http.Cookie{tru.usernameToCookie["2"]}, "0")
  if err != nil {
    t.Errorf("couldn't vote to kick player 0: " + err.Error())
  }

  // Confirm correct amt of players remains
  if len(tru.room.pm.players) != 2 {
    t.Errorf("expected 2 players, got %d", len(tru.room.pm.players))
  }
  // Make sure the correct player got kicked and the remaining players are in
  // the right order
  if tru.room.pm.players[0].username != "1" ||
     tru.room.pm.players[1].username != "2" {
    t.Errorf("expected player usernames [\"1\", \"2\"], got [\"%s\", \"%s\"]",
             tru.room.pm.players[0].username, tru.room.pm.players[1].username)
  }

  // Expected player to move is "1"
  if tru.room.pm.currentPlayerIdx != 0 {
    t.Errorf("expected current player idx to be 0, was %d",
             tru.room.pm.currentPlayerIdx)
  }

  // Now make sure the time remaining is not still 30s
  if tru.room.pm.currentPlayerDeadline == preKickDeadline {
    t.Errorf("kicked player's remaining time spilled over to next player")
  }
}

func TestAffix(t *testing.T) {
  tru := newTestRoomUtils(Config {
    MaxPlayers: 16,
    MinWordLength: 5,
    IsPublic: true,
    EliminationThreshold: 0,
    AllowRepeatWords: false,
    PlayerTimePerWord: time.Second * 0,
  })

  err := tru.addNPlayers(2)
  if err != nil {
    t.Errorf("couldn't add two players: " + err.Error())
  }

  err = tru.readyUpAllPlayers()
  if err != nil {
    t.Errorf("couldn't ready up all players: " + err.Error())
  }

  // Try to affix two letters at once
  err = tru.room.AffixWord(tru.currentPlayerCookies(), "a", "b")
  if err == nil {
    t.Errorf("added both a prefix and suffix")
  }

  // Try to affix two letters at once
  err = tru.room.AffixWord(tru.currentPlayerCookies(), "ab", "")
  if err == nil {
    t.Errorf("added a prefix or len 2")
  }

  err = tru.room.AffixWord(tru.currentPlayerCookies(), "", "ab")
  if err == nil {
    t.Errorf("added a suffix or len 2")
  }

  preAffixPlayer := tru.room.pm.currentPlayerUsername()
  err = tru.room.AffixWord(tru.currentPlayerCookies(), "a", "")
  if err != nil {
    t.Errorf("couldn't add valid prefix: " + err.Error())
  }
  // make sure current player changes after affixing
  if preAffixPlayer == tru.room.pm.currentPlayerUsername() {
    t.Errorf("current player did not increment after affixing letter")
  }

  err = tru.room.AffixWord(tru.currentPlayerCookies(), "", "b")
  if err != nil {
    t.Errorf("couldn't add valid suffix")
  }

  if tru.room.stem != "AB" {
    t.Errorf("expected stem to equal \"ab\", got \"%s\"", tru.room.stem)
  }
}
