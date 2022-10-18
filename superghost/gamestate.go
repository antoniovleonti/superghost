package superghost

import (
  "encoding/json"
  "fmt"
  "net/http"
  "strings"
  "sync"
  "time"
)

type GameMode int
const (
  kEdit GameMode = iota
  kRebut
  kInsufficientPlayers
)
func (p GameMode) String() string {
  switch p {
    case kEdit:
      return "edit"
    case kRebut:
      return "rebut"
    case kInsufficientPlayers:
      return "insufficient players"
    default:
      panic("invalid GameMode value")
  }
}

type GameState struct {
  mutex sync.RWMutex
  players []*Player
  usernameToPlayer map[string]*Player
  word string
  mode GameMode
  nextPlayer int
  lastPlayer string
  firstPlayer int
  lastRoundResult string
}

type JGameState struct { // publicly visible version of gamestate
  Players []*Player       `json:"players"`
  Word string             `json:"word"`
  Mode string             `json:"mode"`
  NextPlayer int          `json:"nextPlayer"`
  LastPlayer string       `json:"lastPlayer"`
  FirstPlayer int         `json:"firstPlayer"`
  LastRoundResult string  `json:"lastRoundResult"`
}

func (gs *GameState) MarshalJSON() ([]byte, error) {
  gs.mutex.RLock()
  defer gs.mutex.RUnlock()

  return json.Marshal(JGameState {
    Players: gs.players,
    Word: gs.word,
    Mode: gs.mode.String(),
    NextPlayer: gs.nextPlayer,
    LastPlayer: gs.lastPlayer,
    FirstPlayer: gs.firstPlayer,
    LastRoundResult: gs.lastRoundResult,
  })
}

func NewGameState() *GameState {
  gs := new(GameState)
  gs.players = make([]*Player, 0)
  gs.usernameToPlayer = make(map[string]*Player)
  gs.mode = kInsufficientPlayers
  gs.nextPlayer = 0
  gs.lastPlayer = ""
  gs.firstPlayer = 0
  return gs
}

func (gs *GameState) getValidCookie(cookies []*http.Cookie) (string, bool) {
  for _, cookie := range cookies {
    if gs.isValidCookie(cookie) {
      return cookie.Name, true
    }
  }
  return "", false
}

func (gs *GameState) addPlayer(
    username string, maxPlayers int) (*http.Cookie, error) {
  gs.mutex.Lock()
  defer gs.mutex.Unlock()
  if len(gs.players) >= maxPlayers {
    return nil, fmt.Errorf("player limit reached")
  }
  // sanitize & validate username
  if !_usernamePattern.MatchString(username) {
    return nil, fmt.Errorf("username must be alphanumeric")
  }
  if _, ok := gs.usernameToPlayer[username]; ok {
    return nil, fmt.Errorf("username '%s' already in use", username)
  }

  p := NewPlayer(username)
  gs.usernameToPlayer[username] = p
  gs.players = append(gs.players, p)
  if len(gs.players) >= 2 && gs.mode == kInsufficientPlayers {
    gs.mode = kEdit
  }
  if len(gs.players) < 2 {
    gs.mode = kInsufficientPlayers
    gs.newRound("")
  }
  return p.cookie, nil
}

func (gs *GameState) challengeIsWord(
    cookies []*http.Cookie, minLength int) error {
  if _, ok := gs.getInTurnCookie(cookies); !ok {
    return fmt.Errorf("it is not your turn")
  }
  gs.mutex.Lock()
  defer gs.mutex.Unlock()

  if gs.mode != kEdit {
    return fmt.Errorf("cannot challenge in %s mode", gs.mode.String())
  }
  if len(gs.word) < minLength {
    return fmt.Errorf("minimum word length not met")
  }

  isWord, err := validateWord(gs.word)
  if err != nil {
    return err
  }
  var loser string
  var correctness string
  if isWord {
    correctness = "correctly"
    loser = gs.lastPlayer
    if p, ok := gs.usernameToPlayer[gs.lastPlayer]; ok {
      p.score++
    }
  } else {
    correctness = "incorrectly"
    p := gs.players[gs.nextPlayer]
    p.score++
    loser = p.username
  }
  lastRoundResult := fmt.Sprintf(
      "%s %s claimed %s spelled a word with '%s'; +1 %s",
      gs.players[gs.nextPlayer].username, correctness,
      gs.lastPlayer, gs.word, loser)
  gs.newRound(lastRoundResult)
  return nil
}

func (gs *GameState) challengeContinuation(cookies []*http.Cookie) error {
  if _, ok := gs.getInTurnCookie(cookies); !ok {
    return fmt.Errorf("it is not your turn")
  }

  gs.mutex.Lock()
  defer gs.mutex.Unlock()

  if gs.mode != kEdit {
    return fmt.Errorf("cannot challenge in %s mode", gs.mode.String())
  }
  if len(gs.word) < 1 {
    return fmt.Errorf("cannot challenge empty stem")
  }

  tmpNextPlayer := gs.nextPlayer
  foundLastPlayer := false
  // make sure the challenged player hasn't left
  for i, p := range gs.players {
    if p.username == gs.lastPlayer {
      foundLastPlayer = true
      gs.nextPlayer = i
    }
  }
  if !foundLastPlayer {
    lastRoundResult := fmt.Sprintf(
        "%s challenged %s, who has left the game",
        gs.lastPlayer, gs.players[gs.nextPlayer].username)
    gs.newRound(lastRoundResult)
  }

  gs.lastPlayer = gs.players[tmpNextPlayer % len(gs.players)].username
  gs.mode = kRebut
  return nil
}

func (gs *GameState) rebutChallenge(
    cookies []*http.Cookie, continuation string, minLength int) error {
  if gs.mode != kRebut {
    return fmt.Errorf("cannot rebut in %s mode", gs.mode.String())
  }
  if _, ok := gs.getInTurnCookie(cookies); !ok {
    return fmt.Errorf("it is not your turn")
  }
  if len(continuation) < minLength {
    return fmt.Errorf("minimum word length not met")
  }

  // check if it is a word
  continuation = strings.TrimSpace(continuation)
  isWord, err := validateWord(continuation)
  if err != nil {
    return err
  }
  // update game state accordingly
  var loser string
  var success string
  if isWord && strings.Contains(continuation, gs.word) {
    // challenger gets a letter
    success = "successfully"
    loser = gs.lastPlayer
    if p, ok := gs.usernameToPlayer[gs.lastPlayer]; ok {
      p.score++
    }
  } else {
    success = "unsuccessfully"
    p := gs.players[gs.nextPlayer]
    p.score++
    loser = p.username
  }
  lastRoundResult := fmt.Sprintf(
      "%s %s rebutted %s's challenge with '%s'; +1 %s",
      gs.players[gs.nextPlayer].username, success,
      gs.lastPlayer, continuation, loser)
  gs.newRound(lastRoundResult)
  return nil
}

func (gs *GameState) affixWord(
    cookies []*http.Cookie, prefix string, suffix string) error {
  gs.mutex.RLock()
  if gs.mode != kEdit {
    gs.mutex.RUnlock()
    return fmt.Errorf("cannot edit word in %s mode", gs.mode.String())
  }
  gs.mutex.RUnlock()
  if _, ok := gs.getInTurnCookie(cookies); !ok {
    return fmt.Errorf("it is not your turn")
  }
  if !_affixPattern.MatchString(prefix + suffix) {
    return fmt.Errorf(
        "exactly one alphabetical prefix OR suffix must be provided")
  }

  gs.mutex.Lock()
  defer gs.mutex.Unlock()

  gs.word = prefix + gs.word + suffix
  gs.lastPlayer = gs.players[gs.nextPlayer].username
  gs.lastRoundResult = ""
  if len(gs.players) == 0 {
    gs.nextPlayer = 0  // Seems extremely unlikely but I'd rather be safe
  } else {
    gs.nextPlayer = (gs.nextPlayer + 1) % len(gs.players)
  }
  return nil
}

func (gs *GameState) heartbeat(cookies []*http.Cookie) error {
  username, ok := gs.getValidCookie(cookies) // needs mutex
  if !ok {
    return fmt.Errorf("no credentials provided")
  }
  gs.mutex.Lock()
  defer gs.mutex.Unlock()

  p, ok := gs.usernameToPlayer[username]
  if !ok {
    return fmt.Errorf("player does not exist")
  }
  p.heartbeat()
  return nil
}

func (gs *GameState) isValidCookie(cookie *http.Cookie) bool {
  gs.mutex.RLock()
  defer gs.mutex.RUnlock()

  if _, ok := gs.usernameToPlayer[cookie.Name]; !ok {
    return false
  }
  return gs.usernameToPlayer[cookie.Name].cookie.Value == cookie.Value
}

func (gs *GameState) getInTurnCookie(cookies []*http.Cookie) (
    *http.Cookie, bool) {
  for _, cookie := range cookies {
    if gs.isInTurnCookie(cookie) {
      return cookie, true
    }
  }
  return nil, false
}

func (gs *GameState) isInTurnCookie(cookie *http.Cookie) bool {
  gs.mutex.RLock()
  defer gs.mutex.RUnlock()

  p := gs.players[gs.nextPlayer % len(gs.players)]
  return (p.username == cookie.Name) && (p.cookie.Value == cookie.Value)
}


func (gs *GameState) newRound(lastRoundResult string) {
  gs.word = ""
  if len(gs.players) == 0 {
    gs.firstPlayer = 0
  } else {
    gs.firstPlayer = (gs.firstPlayer + 1) % len(gs.players)
  }
  gs.lastPlayer = ""
  gs.nextPlayer = gs.firstPlayer
  gs.lastRoundResult = lastRoundResult
  if len(gs.players) >= 2 {
    gs.mode = kEdit
  } else {
    gs.mode = kInsufficientPlayers
  }
}

// returns true if any players are removed, false otherwise
func (gs *GameState) removeDeadPlayers(duration time.Duration) bool {
  gs.mutex.Lock()
  defer gs.mutex.Unlock()

  didRemovePlayer := false
  for i := len(gs.players) - 1; i >= 0; i-- {
    if time.Since(gs.players[i].lastHeartbeat) > duration {
      gs.removePlayer(i)
      didRemovePlayer = true
    }
  }
  if didRemovePlayer && (len(gs.players) < 2) {
    gs.newRound("")
  }
  return didRemovePlayer
}

func (gs *GameState) removePlayer(index int) {
  if index < gs.nextPlayer {
    gs.nextPlayer--
  }
  if index < gs.firstPlayer {
    gs.firstPlayer--
  }

  delete(gs.usernameToPlayer, gs.players[index].username)

  if (index == len(gs.players) - 1) {
    gs.players = gs.players[:index] // avoid out of bounds...
  } else {
    gs.players = append(gs.players[:index], gs.players[index+1:]...)
  }
}

func (gs *GameState) Heartbeat(cookies []*http.Cookie) error {
  username, ok := gs.getValidCookie(cookies) // needs mutex
  if !ok {
    return fmt.Errorf("no credentials provided")
  }
  gs.mutex.Lock()
  defer gs.mutex.Unlock()

  p, ok := gs.usernameToPlayer[username]
  if !ok {
    return fmt.Errorf("player does not exist")
  }
  p.heartbeat()
  return nil
}

