package superghost

import (
  "encoding/json"
  "fmt"
  "net/http"
  "strings"
  "sync"
  "time"
)

type State int
const (
  kEdit State = iota
  kRebut
  kInsufficientPlayers
)
func (p State) String() string {
  switch p {
    case kEdit:
      return "edit"
    case kRebut:
      return "rebut"
    case kInsufficientPlayers:
      return "insufficient players"
    default:
      panic("invalid State value")
  }
}

type Config struct {
  MaxPlayers int
  MinStemLength int
  IsPublic bool
}

type Room struct {
  config *Config
  mutex sync.RWMutex
  players []*Player
  usernameToPlayer map[string]*Player
  stem string
  state State
  nextPlayer int
  lastPlayer string
  firstPlayer int
  lastRoundResult string
  log []string
  logItemsFlushed int
}

type jRoom struct { // publicly visible version of gamestate
  Players []*Player `json:"players"`
  Word string       `json:"word"`
  Awaiting string   `json:"state"`
  NextPlayer int    `json:"nextPlayer"`
  LastPlayer string `json:"lastPlayer"`
  FirstPlayer int   `json:"firstPlayer"`
  LogFlush []string `json:"logFlush"`
}

func (gs *Room) MarshalJSON() ([]byte, error) {
  gs.mutex.RLock()
  defer gs.mutex.RUnlock()

  return json.Marshal(jRoom {
    Players: gs.players,
    Word: strings.ToUpper(gs.stem),
    Awaiting: gs.state.String(),
    NextPlayer: gs.nextPlayer,
    LastPlayer: gs.lastPlayer,
    FirstPlayer: gs.firstPlayer,
    LogFlush: gs.log[gs.logItemsFlushed:],
  })
}

func (gs *Room) MarshalJSONFullLog() ([]byte, error) {
  gs.mutex.RLock()
  defer gs.mutex.RUnlock()

  return json.Marshal(jRoom {
    Players: gs.players,
    Word: strings.ToUpper(gs.stem),
    Awaiting: gs.state.String(),
    NextPlayer: gs.nextPlayer,
    LastPlayer: gs.lastPlayer,
    FirstPlayer: gs.firstPlayer,
    LogFlush: gs.log,
  })
}

func NewRoom(config Config) *Room {
  gs := new(Room)

  gs.config = new(Config)
  gs.config.MaxPlayers = config.MaxPlayers
  gs.config.MinStemLength = config.MinStemLength
  gs.config.IsPublic = config.IsPublic

  gs.players = make([]*Player, 0)
  gs.usernameToPlayer = make(map[string]*Player)
  gs.state = kInsufficientPlayers
  gs.lastPlayer = ""
  gs.logItemsFlushed = 0
  gs.log = make([]string, 0)
  return gs
}

// public, mutex-protected version
func (gs *Room) GetValidCookie(cookies []*http.Cookie) (string, bool) {
  gs.mutex.RLock()
  defer gs.mutex.RUnlock()
  return gs.getValidCookie(cookies)
}

func (gs *Room) getValidCookie(cookies []*http.Cookie) (string, bool) {
  for _, cookie := range cookies {
    if gs.isValidCookie(cookie) {
      return cookie.Name, true
    }
  }
  return "", false
}

func (gs *Room) AddPlayer(username string, path string) (*http.Cookie, error) {
  gs.mutex.Lock()
  defer gs.mutex.Unlock()

  if len(gs.players) >= gs.config.MaxPlayers {
    return nil, fmt.Errorf("player limit reached")
  }
  if !_usernamePattern.MatchString(username) {
    return nil, fmt.Errorf("username must be alphanumeric")
  }
  if _, ok := gs.usernameToPlayer[username]; ok {
    return nil, fmt.Errorf("username '%s' already in use", username)
  }

  gs.logItemsFlushed = len(gs.log)

  p := NewPlayer(username, path)
  gs.usernameToPlayer[username] = p
  gs.players = append(gs.players, p)

  gs.log = append(gs.log, fmt.Sprintf("<i>%s</i> joined the game!", p.username))

  if len(gs.players) >= 2 && gs.state == kInsufficientPlayers {
    gs.state = kEdit
  }
  if len(gs.players) < 2 {
    gs.state = kInsufficientPlayers
  }
  return p.cookie, nil
}

func (gs *Room) ChallengeIsWord(cookies []*http.Cookie) error {
  gs.mutex.Lock()
  defer gs.mutex.Unlock()

  if _, ok := gs.getInTurnCookie(cookies); !ok {
    return fmt.Errorf("it is not your turn")
  }
  if gs.state != kEdit {
    return fmt.Errorf("cannot challenge right now")
  }
  if len(gs.stem) < gs.config.MinStemLength {
    return fmt.Errorf("minimum word length not met")
  }

  gs.logItemsFlushed = len(gs.log)
  gs.log = append(gs.log, fmt.Sprintf(
      "<i>%s</i> claimed <i>%s</i> spelled a word.",
      gs.players[gs.nextPlayer].username, gs.lastPlayer,
      strings.ToUpper(gs.stem)))

  isWord, err := validateWord(gs.stem)
  if err != nil {
    return err
  }
  var loser string
  var isOrIsNot string
  if isWord {
    loser = gs.lastPlayer
    isOrIsNot = "IS"
    if p, ok := gs.usernameToPlayer[gs.lastPlayer]; ok {
      p.score++
    }
  } else {
    isOrIsNot = "IS NOT"
    p := gs.players[gs.nextPlayer]
    p.score++
    loser = p.username
  }
  gs.log = append(gs.log, fmt.Sprintf("'%s' %s a word! +1 <i>%s</i>.",
                                      strings.ToUpper(gs.stem), isOrIsNot,
                                      loser))
  gs.newRound()
  return nil
}

func (gs *Room) ChallengeContinuation(cookies []*http.Cookie) error {
  gs.mutex.Lock()
  defer gs.mutex.Unlock()

  if _, ok := gs.getInTurnCookie(cookies); !ok {
    return fmt.Errorf("it is not your turn")
  }
  if gs.state != kEdit {
    return fmt.Errorf("cannot challenge right now")
  }
  if len(gs.stem) < 1 {
    return fmt.Errorf("cannot challenge empty stem")
  }

  gs.logItemsFlushed = len(gs.log)

  // make sure the challenged player hasn't left
  lastPlayerIdx := -1
  for i, p := range gs.players {
    if p.username == gs.lastPlayer {
      lastPlayerIdx = i
      break
    }
  }
  if lastPlayerIdx == -1 {
    gs.log = append(gs.log, fmt.Sprintf(
        "<i>%s</i> challenged <i>%s</i>, who left the game.",
        gs.players[gs.nextPlayer].username, gs.lastPlayer))
    gs.newRound()
    return nil
  }
  gs.log = append(gs.log, fmt.Sprintf(
      "<i>%s</i> challenged <i>%s</i> for a continuation.",
      gs.players[gs.nextPlayer].username, gs.lastPlayer))

  gs.lastPlayer = gs.players[gs.nextPlayer].username
  if len(gs.players) == 0 {
    gs.nextPlayer = 0
  } else {
    gs.nextPlayer = (gs.nextPlayer + 1) % len(gs.players)
  }
  gs.state = kRebut
  return nil
}

func (gs *Room) RebutChallenge(cookies []*http.Cookie,
                                prefix string,
                                suffix string) error {
  gs.mutex.Lock()
  defer gs.mutex.Unlock()

  if gs.state != kRebut {
    return fmt.Errorf("cannot rebut right now")
  }
  if _, ok := gs.getInTurnCookie(cookies); !ok {
    return fmt.Errorf("it is not your turn")
  }

  continuation := strings.ToUpper(prefix + gs.stem + suffix)
  if len(continuation) < gs.config.MinStemLength {
    return fmt.Errorf("minimum word length not met")
  }

  gs.logItemsFlushed = len(gs.log)
  gs.log = append(gs.log, fmt.Sprintf("<i>%s</i> rebutted with '%s'.",
                                      gs.players[gs.nextPlayer].username,
                                      continuation))
  // check if it is a word
  isWord, err := validateWord(continuation)
  if err != nil {
    return err
  }
  // update game Room accordingly
  var loser string
  var isOrIsNot string
  if isWord {
    // challenger gets a letter
    isOrIsNot = "IS"
    loser = gs.lastPlayer
    if p, ok := gs.usernameToPlayer[gs.lastPlayer]; ok {
      p.score++
    }
  } else {
    isOrIsNot = "IS NOT"
    p := gs.players[gs.nextPlayer]
    p.score++
    loser = p.username
  }
  gs.log = append(gs.log, fmt.Sprintf("'%s' %s a word! +1 <i>%s</i>.",
                                      continuation, isOrIsNot, loser))
  gs.newRound()
  return nil
}

func (gs *Room) AffixWord(
    cookies []*http.Cookie, prefix string, suffix string) error {
  gs.mutex.Lock()
  defer gs.mutex.Unlock()
  if gs.state != kEdit {
    return fmt.Errorf("cannot affix right now")
  }
  if _, ok := gs.getInTurnCookie(cookies); !ok {
    return fmt.Errorf("it is not your turn")
  }
  if !_alphaPattern.MatchString(prefix + suffix) {
    return fmt.Errorf(
        "exactly one alphabetical prefix OR suffix must be provided " +
        "(received: {prefix: '%s', suffix: '%s'})", prefix, suffix)
  }

  // update log
  gs.logItemsFlushed = len(gs.log)
  var affixed string
  if len(prefix) > 0 {
    affixed = "<b>" + prefix + "</b>" + gs.stem
  } else {
    affixed = gs.stem + "<b>" + suffix + "</b>"
  }
  gs.log = append(gs.log, fmt.Sprintf(
      "<i>%s</i>: %s",
      gs.players[gs.nextPlayer].username, strings.ToUpper(affixed)))

  gs.stem = prefix + gs.stem + suffix

  gs.lastPlayer = gs.players[gs.nextPlayer].username
  if len(gs.players) == 0 {
    gs.nextPlayer = 0  // Seems extremely unlikely but I'd rather be safe
  } else {
    gs.nextPlayer = (gs.nextPlayer + 1) % len(gs.players)
  }
  return nil
}

func (gs *Room) isValidCookie(cookie *http.Cookie) bool {
  if _, ok := gs.usernameToPlayer[cookie.Name]; !ok {
    return false
  }
  return gs.usernameToPlayer[cookie.Name].cookie.Value == cookie.Value
}

func (gs *Room) getInTurnCookie(cookies []*http.Cookie) (
    *http.Cookie, bool) {
  for _, cookie := range cookies {
    if gs.isInTurnCookie(cookie) {
      return cookie, true
    }
  }
  return nil, false
}

func (gs *Room) isInTurnCookie(cookie *http.Cookie) bool {
  p := gs.players[gs.nextPlayer % len(gs.players)]
  return (p.username == cookie.Name) && (p.cookie.Value == cookie.Value)
}

func (gs *Room) newRound() {
  gs.stem = ""
  if len(gs.players) == 0 {
    gs.firstPlayer = 0
  } else {
    gs.firstPlayer = (gs.firstPlayer + 1) % len(gs.players)
  }
  gs.lastPlayer = ""
  gs.nextPlayer = gs.firstPlayer
  if len(gs.players) >= 2 {
    gs.state = kEdit
  } else {
    gs.state = kInsufficientPlayers
  }
}

// returns true if any players are removed, false otherwise
func (gs *Room) RemoveDeadPlayers(duration time.Duration) bool {
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
    gs.newRound()
  }
  return didRemovePlayer
}

func (gs *Room) removePlayer(index int) error {
  if index > len(gs.players) {
    return fmt.Errorf("index out of bounds")
  }
  if index < gs.nextPlayer {
    gs.nextPlayer--
  }
  if index < gs.firstPlayer {
    gs.firstPlayer--
  }

  gs.log = append(gs.log, fmt.Sprintf("<i>%s</i> left the game.",
                                      gs.players[index].username))
  delete(gs.usernameToPlayer, gs.players[index].username)

  if (index == len(gs.players) - 1) {
    gs.players = gs.players[:index] // avoid out of bounds...
  } else {
    gs.players = append(gs.players[:index], gs.players[index+1:]...)
  }
  return nil
}

func (gs *Room) Leave(cookies []*http.Cookie) error {
  gs.mutex.Lock()
  defer gs.mutex.Unlock()

  username, ok := gs.getValidCookie(cookies)
  if !ok {
    return fmt.Errorf("no credentials provided")
  }

  gs.logItemsFlushed = len(gs.log)

  for i, p := range gs.players { // we have to find the index of the player
    if p.username == username {
      return gs.removePlayer(i)
    }
  }
  return fmt.Errorf("unexpected error: player not found")
}

func (gs *Room) Heartbeat(cookies []*http.Cookie) error {
  gs.mutex.Lock()
  defer gs.mutex.Unlock()

  username, ok := gs.getValidCookie(cookies) // needs mutex
  if !ok {
    return fmt.Errorf("no credentials provided")
  }

  p, ok := gs.usernameToPlayer[username]
  if !ok {
    return fmt.Errorf("player does not exist")
  }
  p.heartbeat()
  return nil
}

func (gs *Room) Concede(cookies []*http.Cookie) error {
  gs.mutex.Lock()
  defer gs.mutex.Unlock()

  username, ok := gs.getValidCookie(cookies)
  if !ok {
    return fmt.Errorf("no credentials provided")
  }
  switch gs.state {

    case kInsufficientPlayers:
      return fmt.Errorf("cannot concede right now")

    case kEdit:
      if len(gs.stem) == 0 {
        return fmt.Errorf("cannot concede when word is empty")
      }

    case kRebut:
      if (username != gs.lastPlayer &&
          username != gs.players[gs.nextPlayer].username) {
        return fmt.Errorf("it is not your turn")
      }
  }
  gs.logItemsFlushed = len(gs.log)

  gs.usernameToPlayer[username].score++
  gs.log = append(gs.log, fmt.Sprintf(
      "<i>%s</i> conceded the round. +1 <i>%s</i>", username, username))
  gs.newRound()
  return nil
}

func (gs *Room) Votekick(cookies []*http.Cookie, usernameToKick string) error {
  gs.mutex.Lock()
  defer gs.mutex.Unlock()

  voter, ok := gs.getValidCookie(cookies)
  if !ok {
    return fmt.Errorf("no credentials provided")
  }

  playerToKick, ok := gs.usernameToPlayer[usernameToKick]
  if !ok {
    return fmt.Errorf("player not found");
  }

  err := playerToKick.votekick(voter)
  if err != nil {
    return err
  }

  gs.logItemsFlushed = len(gs.log)
  gs.log = append(gs.log, fmt.Sprintf("<i>%s</i> voted to kick <i>%s</i>.",
                                      voter, usernameToKick))
  // if a majority has voted to kick the player, remove them from the game
  if float64(playerToKick.numVotesToKick) >=
     float64(len(gs.players)) / 1.9 {
    for i, p := range gs.players {
      if p.username == usernameToKick {
        gs.removePlayer(i)
        break
      }
    }
  }
  return nil
}

