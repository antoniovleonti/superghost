package superghost

import (
  "encoding/json"
  "errors"
  "fmt"
  "net/http"
  "strings"
  "sync"
  "time"
  // "regexp"
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
}

type JGameState struct { // publicly visible version of gamestate
  Players []*Player  `json:"players"`
  Word string       `json:"word"`
  Mode string   `json:"phase"`
  NextPlayer int   `json:"nextPlayer"`
  LastPlayer string `json:"lastPlayer"`
  FirstPlayer int  `json:"firstPlayer"`
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

func (gs *GameState) AffixWord(
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
  if len(prefix) > 0 && len(suffix) > 0 {
    return fmt.Errorf("only one of prefix or suffix should be specified")
  }
  if len(prefix) != 1 && len(suffix) != 1 {
    return fmt.Errorf("affix must be of length 1")
  }
  if !_lowerPattern.MatchString(prefix) && !_lowerPattern.MatchString(suffix) {
    return fmt.Errorf("affix must be lowercase alphabetic (no unicode)")
  }

  gs.mutex.Lock()
  defer gs.mutex.Unlock()

  gs.word = prefix + gs.word + suffix
  gs.lastPlayer = gs.players[gs.nextPlayer].username
  if len(gs.players) == 0 {
    gs.nextPlayer = 0 // Probably shouldn't be possible but just to be safe
  } else {
    gs.nextPlayer = (gs.nextPlayer + 1) % len(gs.players)
  }
  return nil
}

func (gs *GameState) GetValidCookie(cookies []*http.Cookie) (string, bool) {
  for _, cookie := range cookies {
    if gs.isValidCookie(cookie) {
      return cookie.Name, true
    }
  }
  return "", false
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

func (gs *GameState) AddPlayer(username string) (*http.Cookie, error) {
  gs.mutex.Lock()
  defer gs.mutex.Unlock()
  // sanitize & validate username
  match := _alphanumPattern.MatchString(username)
  if !match {
    return nil, errors.New("username must be alphanumeric")
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
    gs.newRound()
  }
  return p.cookie, nil
}

func (gs *GameState) newRound() {
  gs.word = ""
  if len(gs.players) == 0 {
    gs.firstPlayer = 0
  } else {
    gs.firstPlayer = gs.firstPlayer + 1 % len(gs.players)
  }
  gs.lastPlayer = ""
  gs.nextPlayer = gs.firstPlayer
  if len(gs.players) >= 2 {
    gs.mode = kEdit
  } else {
    gs.mode = kInsufficientPlayers
  }
}

func (gs *GameState) ChallengeIsWord(cookies []*http.Cookie) error {
  if _, ok := gs.getInTurnCookie(cookies); !ok {
    return fmt.Errorf("it is not your turn")
  }
  gs.mutex.Lock()
  defer gs.mutex.Unlock()

  if gs.mode != kEdit {
    return fmt.Errorf("cannot challenge in %s mode", gs.mode.String())
  }

  isWord, err := validateWord(gs.word)
  if err != nil {
    return err
  }
  if isWord {
    if p, ok := gs.usernameToPlayer[gs.lastPlayer]; ok {
      p.score++
    }
  } else {
    gs.players[gs.nextPlayer % len(gs.players)].score++
  }
  gs.newRound()
  return nil
}

func (gs *GameState) ChallengeContinuation(cookies []*http.Cookie) error {
  gs.mutex.Lock()
  defer gs.mutex.Unlock()

  if _, ok := gs.getInTurnCookie(cookies); !ok {
    return fmt.Errorf("it is not your turn")
  }
  if gs.mode != kEdit {
    return fmt.Errorf("cannot challenge in %s mode", gs.mode.String())
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
    gs.newRound()
  }

  gs.lastPlayer = gs.players[tmpNextPlayer % len(gs.players)].username
  gs.mode = kRebut
  return nil
}

func (gs *GameState) RebutChallenge(
    cookies []*http.Cookie, continuation string) error {
  if gs.mode != kRebut {
    return fmt.Errorf("cannot rebut in %s mode", gs.mode.String())
  }
  if _, ok := gs.getInTurnCookie(cookies); !ok {
    return fmt.Errorf("it is not your turn")
  }
  continuation = strings.TrimSpace(continuation)
  if !strings.Contains(continuation, gs.word) {
    return errors.New("continuation must contain current substring")
  }
  // check if it is a word
  isWord, err := validateWord(continuation)
  if err != nil {
    return err
  }
  // update game state accordingly
  if isWord {
    // challenger gets a letter
    if p, ok := gs.usernameToPlayer[gs.lastPlayer]; ok {
      p.score++
    }
  } else {
    gs.players[gs.nextPlayer % len(gs.players)].score++
  }
  gs.newRound()
  return nil
}

// returns true if any players are removed, false otherwise
func (gs *GameState) RemoveDeadPlayers(duration time.Duration) bool {
  gs.mutex.Lock()
  defer gs.mutex.Unlock()

  didRemovePlayer := false
  for i := len(gs.players) - 1; i >= 0; i-- {
    if time.Since(gs.players[i].lastHeartbeat) > duration {
      fmt.Println(time.Since(gs.players[i].lastHeartbeat))
      gs.removePlayer(i)
      didRemovePlayer = true
    }
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
  fmt.Println(gs.players[index].username)

  delete(gs.usernameToPlayer, gs.players[index].username)

  if (index == len(gs.players) - 1) {
    gs.players = gs.players[:index] // avoid out of bounds...
  } else {
    gs.players = append(gs.players[:index], gs.players[index+1:]...)
  }
}

func (gs *GameState) Heartbeat(cookies []*http.Cookie) error {
  username, ok := gs.GetValidCookie(cookies) // needs mutex
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

