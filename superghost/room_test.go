package superghost

import (
  "testing"
  "time"
  "net/http"
  "strconv"
)

type testRoomUtils struct {
  room *Room
  asyncUpdateCh chan struct{}
  usernameToCookie map[string]*http.Cookie
}

func newTestRoomUtils(config Config) *testRoomUtils {
  tru := new(testRoomUtils)
  tru.asyncUpdateCh = make(chan struct{})
  tru.room = NewRoom(config, tru.asyncUpdateCh)
  tru.usernameToCookie = make(map[string]*http.Cookie)
  return tru
}

func newDefaultTimedNoEliminationTestRoomUtils() *testRoomUtils {
  return newTestRoomUtils(Config {
    MaxPlayers: 16,
    MinWordLength: 5,
    IsPublic: true,
    EliminationThreshold: 0,
    AllowRepeatWords: false,
    PlayerTimePerWord: time.Second * 60,
  })
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

func (tru *testRoomUtils) currentPlayerCookies() []*http.Cookie {
  return []*http.Cookie{tru.room.pm.currentPlayer().cookie}
}

func TestGameEndsWhenOnePlayerRemains(t *testing.T) {
}

func TestNoDeadlineAtRoundStart(t *testing.T) {
  tru := newDefaultTimedNoEliminationTestRoomUtils()
  err := tru.addNPlayers(2)
  if err != nil {
    t.Errorf("couldn't add players: " + err.Error())
  }

  // Deadline should be zeroed aka no deadline
  if tru.room.pm.doesDeadlineExist() {
    t.Errorf("deadline should not be set before first player makes a move")
  }
}

func TestDeadlineExistsAfterFirstMove(t *testing.T) {
  tru := newDefaultTimedNoEliminationTestRoomUtils()
  err := tru.addNPlayers(2)
  if err != nil {
    t.Errorf("couldn't add players: " + err.Error())
  }

  err = tru.room.AffixLetter(tru.currentPlayerCookies(), "", "b")
  if err != nil {
    t.Errorf(err.Error())
  }

  // First player made a move, deadline should be set now.
  if !tru.room.pm.doesDeadlineExist() {
    t.Errorf("deadline should be set after first move of round")
  }
}

func TestDeadlineWipedAfterRoundEndAndGameEnd(t *testing.T) {
  tru := newTestRoomUtils(Config {
    MaxPlayers: 16,
    MinWordLength: 5,
    IsPublic: true,
    EliminationThreshold: 0,
    AllowRepeatWords: false,
    PlayerTimePerWord: time.Second * 60,
  })
  err := tru.addNPlayers(2)
  if err != nil {
    t.Errorf("couldn't add players: " + err.Error())
  }

  // Affix, concede until the game is over
  for tru.room.pm.players[0].score = 1; tru.room.pm.allScoresAreZero(); {
    if tru.room.pm.doesDeadlineExist() {
      t.Errorf("deadline should not exist at the beginning of a round")
    }

    err = tru.room.AffixLetter(tru.currentPlayerCookies(), "", "b")
    if err != nil {
      t.Errorf(err.Error())
    }
    err = tru.room.Concede(tru.currentPlayerCookies())
    if err != nil {
      t.Errorf(err.Error())
    }
  }

  if tru.room.pm.doesDeadlineExist() {
    t.Errorf("deadline should not exist at the beginning of new game")
  }
}

func TestVotekickCurrentPlayer(t *testing.T) {
  tru := newDefaultTimedNoEliminationTestRoomUtils()
  err := tru.addNPlayers(3)
  if err != nil {
    t.Errorf("couldn't add players: " + err.Error())
  }

  // Edit the first player's time to make sure when they are kicked their
  // remaining time does not spill over to the next person (bug as of time of
  // writing this test). Time starts at 60.
  tru.room.pm.currentPlayer().timeRemaining = 30 * time.Second

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
  if tru.room.pm.currentPlayerDeadline != preKickDeadline {
    t.Errorf("kicked player's remaining time spilled over to next player")
  }
}

func TestAffix(t *testing.T) {
  tru := newDefaultTimedNoEliminationTestRoomUtils()
  err := tru.addNPlayers(2)
  if err != nil {
    t.Errorf("couldn't add two players: " + err.Error())
  }

  // Try to affix two letters at once
  err = tru.room.AffixLetter(tru.currentPlayerCookies(), "a", "b")
  if err == nil {
    t.Errorf("added both a prefix and suffix")
  }

  // Try to affix two letters at once
  err = tru.room.AffixLetter(tru.currentPlayerCookies(), "ab", "")
  if err == nil {
    t.Errorf("added a prefix or len 2")
  }

  err = tru.room.AffixLetter(tru.currentPlayerCookies(), "", "ab")
  if err == nil {
    t.Errorf("added a suffix or len 2")
  }

  preAffixPlayer := tru.room.pm.currentPlayerUsername()
  err = tru.room.AffixLetter(tru.currentPlayerCookies(), "a", "")
  if err != nil {
    t.Errorf("couldn't add valid prefix: " + err.Error())
  }
  // make sure current player changes after affixing
  if preAffixPlayer == tru.room.pm.currentPlayerUsername() {
    t.Errorf("current player did not increment after affixing letter")
  }

  err = tru.room.AffixLetter(tru.currentPlayerCookies(), "", "b")
  if err != nil {
    t.Errorf("couldn't add valid suffix")
  }

  if tru.room.stem != "AB" {
    t.Errorf("expected stem to equal \"ab\", got \"%s\"", tru.room.stem)
  }
}

func TestCancellableLeave(t *testing.T) {
  tru := newDefaultTimedNoEliminationTestRoomUtils()
  err := tru.addNPlayers(3)
  if err != nil {
    t.Errorf("couldn't add two players: " + err.Error())
  }

  username := "0"
  player, ok := tru.room.pm.usernameToPlayer[username]
  if !ok {
    t.Errorf("couldn't find player 0")
  }
  cookies := []*http.Cookie{player.cookie}

  // Leave then cancel, then make sure it actually got cancelled
  tru.room.ScheduleLeave(cookies)
  // make sure leave is scheduled
  if _, ok := tru.room.usernameToCancelLeaveCh[username]; !ok {
    t.Errorf("cancel leave channel does not exist")
  }
  tru.room.CancelLeaveIfScheduled(cookies)
  time.Sleep(10 * time.Millisecond)
  if _, ok := tru.room.usernameToCancelLeaveCh[username]; ok {
    t.Errorf("cancel leave channel was not deleted after cancel")
  }
  // Wait and see if player leaves
  deadline := time.NewTimer(1 * time.Second)
  select {
    case <-deadline.C:
      // Just escape the select statement. Success.
    case <-tru.asyncUpdateCh:
      t.Errorf("player left even though CancelLeaveIfScheduled was called")
  }

  // Leave and don't cancel, then make sure the player eventually leaves
  tru.room.ScheduleLeave(cookies)
  // make sure a channel exists to cancel
  if _, ok := tru.room.usernameToCancelLeaveCh[username]; !ok {
    t.Errorf("cancel leave channel does not exist (2)")
  }
  // Wait and see if player leaves
  deadline = time.NewTimer(10 * time.Second)
  select {
    case <-deadline.C:
      t.Errorf("scheduled leave did not take effect (or async update channel " +
               "was not notified if it did)")
    case <-tru.asyncUpdateCh:
      // Verify player count is correct
      if len(tru.room.pm.players) != 2 {
        t.Errorf("got async update but player count is %d (expected 2)",
                 len(tru.room.pm.players))
      }
      if _, ok := tru.room.pm.usernameToPlayer[username]; ok {
        t.Errorf("got async update, but player 0 is still in the game")
      }
  }

  if _, ok := tru.room.usernameToCancelLeaveCh[username]; ok {
    t.Errorf("cancel leave channel was not deleted after player left")
  }
}

func TestCancellableLeaveToOnePlayerEndsGame(t *testing.T) {
  tru := newDefaultTimedNoEliminationTestRoomUtils()
  err := tru.addNPlayers(2)
  if err != nil {
    t.Errorf("couldn't add two players: " + err.Error())
  }

  username := "0"
  player, ok := tru.room.pm.usernameToPlayer[username]
  if !ok {
    t.Errorf("couldn't find player 0")
  }
  cookies := []*http.Cookie{player.cookie}

  // Leave then cancel, then make sure it actually got cancelled
  tru.room.ScheduleLeave(cookies)
  // make sure leave is scheduled
  if _, ok := tru.room.usernameToCancelLeaveCh[username]; !ok {
    t.Errorf("cancel leave channel does not exist")
  }

  // Leave and don't cancel, then make sure the player eventually leaves
  tru.room.ScheduleLeave(cookies)
  // make sure a channel exists to cancel
  if _, ok := tru.room.usernameToCancelLeaveCh[username]; !ok {
    t.Errorf("cancel leave channel does not exist")
  }

  // Wait and see if player leaves
  deadline := time.NewTimer(1 * time.Second)
  select {
    case <-deadline.C:
      t.Errorf("scheduled leave did not take effect (or async update channel " +
               "was not notified if it did)")
    case <-tru.asyncUpdateCh:
      // Verify player count is correct
      if len(tru.room.pm.players) != 1 {
        t.Errorf("got async update but player count is %d (expected 1)",
                 len(tru.room.pm.players))
      }
      if _, ok := tru.room.pm.usernameToPlayer[username]; ok {
        t.Errorf("got async update, but player 0 is still in the game")
      }
  }

  // Now see if game was ended
  if logItemType := tru.room.log.history[len(tru.room.log.history)-1].Type;
      logItemType != kInsufficientPlayers {
    t.Errorf("last log item = %s (expected %s)",
             logItemType, kInsufficientPlayers)
  }
}

func TestGameLoop(t *testing.T) {
  tru := newDefaultTimedNoEliminationTestRoomUtils()
  err := tru.addNPlayers(2)
  if err != nil {
    t.Errorf("couldn't add two players: " + err.Error())
  }

  // Try to affix two letters at once
  err = tru.room.AffixLetter(tru.currentPlayerCookies(), "", "s")
  if err != nil {
    t.Errorf("couldn't affix (1)")
  }

  err = tru.room.AffixLetter(tru.currentPlayerCookies(), "", "t")
  if err != nil {
    t.Errorf("couldn't affix (2)`")
  }

  err = tru.room.ChallengeContinuation(tru.currentPlayerCookies())
  if err != nil {
    t.Errorf("couldn't challenge continuation")
  }

  err = tru.room.RebutChallenge(tru.currentPlayerCookies(), "te", "ing")
  if err != nil {
    t.Errorf("couldn't rebut challenge")
  }
}

func TestPlayerTimeRemainingAfterFirstMove(t *testing.T) {
  tru := newDefaultTimedNoEliminationTestRoomUtils()
  err := tru.addNPlayers(2)
  if err != nil {
    t.Errorf("couldn't add two players: " + err.Error())
  }

  err = tru.room.AffixLetter(tru.currentPlayerCookies(), "", "s")
  if err != nil {
    t.Errorf("couldn't affix")
  }

  if tru.room.pm.players[0].timeRemaining < 0 {
    t.Errorf("time remaining is less than zero!")
  }
}
