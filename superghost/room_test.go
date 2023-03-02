package superghost

import (
  "github.com/stretchr/testify/assert"
  "net/http"
  "strconv"
  "testing"
  "time"
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

func (tru *testRoomUtils) getCookiesFromPlayerIdx(idx int) []*http.Cookie {
  return []*http.Cookie{tru.room.pm.players[idx].cookie}
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

func TestHostCanKick(t *testing.T) {
  tru := newDefaultTimedNoEliminationTestRoomUtils()
  err := tru.addNPlayers(2)
  if err != nil {
    t.Errorf("couldn't add players: " + err.Error())
  }

  kickRecipientUsername := tru.room.pm.players[1].username
  err = tru.room.Kick(tru.getCookiesFromPlayerIdx(0), kickRecipientUsername)
  if err != nil {
    t.Errorf(err.Error())
  }
  if len(tru.room.pm.players) != 1 {
    t.Errorf("expected 1 player after successful kick, got %d",
             len(tru.room.pm.players))
  }
  // Check the log message is correct
  message := tru.room.log.history[len(tru.room.log.history)-1]
  assert.Equal(t, kKick, message.Type)
  assert.Equal(t, tru.room.pm.players[0].username, message.From)
  assert.Equal(t, kickRecipientUsername, message.To)
}

func TestOnlyHostCanKick(t *testing.T) {
  tru := newDefaultTimedNoEliminationTestRoomUtils()
  err := tru.addNPlayers(2)
  if err != nil {
    t.Errorf("couldn't add players: " + err.Error())
  }

  err = tru.room.Kick(tru.getCookiesFromPlayerIdx(1),
                      tru.room.pm.players[0].username)
  if err == nil {
    t.Errorf("expected kick from non-host to fail")
  }
  if len(tru.room.pm.players) != 2 {
    t.Errorf("expected 2 players after failed kick, got %d",
             len(tru.room.pm.players))
  }
}

func TestAffix(t *testing.T) {
  tru := newDefaultTimedNoEliminationTestRoomUtils()
  err := tru.addNPlayers(2)
  if err != nil {
    t.Errorf("couldn't add two players: " + err.Error())
  }

  badCases := [][]string{
    {"a", "b"},
    {"ab", ""},
    {"", "ab"},
    {"", ""},
  }
  for _, pair := range badCases {
    assert.Error(t, tru.room.AffixLetter(tru.currentPlayerCookies(),
                                         pair[0], pair[1]))
    assert.Zero(t, tru.room.turnID)  // Should not increment on err
  }

  preAffixPlayer := tru.room.pm.currentPlayerUsername()

  // valid affix
  assert.NoError(t, tru.room.AffixLetter(tru.currentPlayerCookies(), "a", ""))
  assert.NotEqual(t, preAffixPlayer, tru.room.pm.currentPlayerUsername())
  assert.Equal(t, 1, tru.room.turnID)
  // valid affix
  assert.NoError(t, tru.room.AffixLetter(tru.currentPlayerCookies(), "", "b"))
  assert.Equal(t, 2, tru.room.turnID)
  // check final result
  assert.Equal(t, "AB", tru.room.stem)
}

func TestCancellableLeave(t *testing.T) {
  tru := newDefaultTimedNoEliminationTestRoomUtils()
  assert.NoError(t, tru.addNPlayers(2))

  username := tru.room.pm.players[0].username
  cookies := tru.getCookiesFromPlayerIdx(0)

  // Leave then cancel, then make sure it actually got cancelled
  tru.room.ScheduleLeave(cookies)
  // make sure leave is scheduled
  assert.Contains(t, tru.room.usernameToCancelLeaveCh, username)

  tru.room.CancelLeaveIfScheduled(cookies)
  time.Sleep(1 * time.Millisecond)

  // If the player is still there and the channel was deleted, I see no way
  // the player could still end up leaving.
  assert.NotContains(t, tru.room.usernameToCancelLeaveCh, username)
  assert.Equal(t, 2, len(tru.room.pm.players))

  // Leave and don't cancel, then make sure the player eventually leaves
  tru.room.ScheduleLeave(cookies)
  // make sure a channel exists to cancel
  assert.Contains(t, tru.room.usernameToCancelLeaveCh, username)
  // Wait and see if player leaves
  deadline := time.NewTimer(1 * time.Second)
  select {
    case <-deadline.C:
      t.Errorf("scheduled leave did not take effect (or async update channel " +
               "was not notified if it did)")
    case <-tru.asyncUpdateCh:
  }
  // Verify player count is correct
  assert.Equal(t, 1, len(tru.room.pm.players))
  assert.NotContains(t, tru.room.pm.usernameToPlayer, username)
  assert.NotContains(t, tru.room.usernameToCancelLeaveCh, username)
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

func TestTurnIDIsZeroAtStart(t *testing.T) {
  tru := newDefaultTimedNoEliminationTestRoomUtils()
  err := tru.addNPlayers(2)
  if err != nil {
    t.Errorf("couldn't add two players: " + err.Error())
  }

  assert.Equal(t, 0, tru.room.turnID)
}
