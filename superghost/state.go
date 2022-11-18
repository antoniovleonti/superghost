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

  currentPlayerIdx int
  currentPlayerUsername string
  lastPlayerUsername string
  startingPlayerIdx int

  stem string
  state State

  log []string
  logItemsPushed int  // The number of log items already sent to clients
}

type jRoom struct { // publicly visible version of gamestate
  Players []*Player `json:"players"`
  Stem string       `json:"stem"`
  State string   `json:"state"`
  CurrentPlayerIdx int    `json:"currentPlayerIdx"`
  LastPlayerUsername string `json:"lastPlayerUsername"`
  StartingPlayerIdx int   `json:"startingPlayerIdx"`
  LogPush []string `json:"logPush"`
}

func (gs *Room) MarshalJSON() ([]byte, error) {
  gs.mutex.RLock()
  defer gs.mutex.RUnlock()

  return json.Marshal(jRoom {
    Players: gs.players,
    Stem: strings.ToUpper(gs.stem),
    State: gs.state.String(),
    CurrentPlayerIdx: gs.currentPlayerIdx,
    LastPlayerUsername: gs.lastPlayerUsername,
    StartingPlayerIdx: gs.startingPlayerIdx,
    LogPush: gs.log[gs.logItemsPushed:],
  })
}

func (gs *Room) MarshalJSONFullLog() ([]byte, error) {
  gs.mutex.RLock()
  defer gs.mutex.RUnlock()

  return json.Marshal(jRoom {
    Players: gs.players,
    Stem: strings.ToUpper(gs.stem),
    State: gs.state.String(),
    CurrentPlayerIdx: gs.currentPlayerIdx,
    LastPlayerUsername: gs.lastPlayerUsername,
    StartingPlayerIdx: gs.startingPlayerIdx,
    LogPush: gs.log,
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
  gs.lastPlayerUsername = ""
  gs.logItemsPushed = 0
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

  gs.logItemsPushed = len(gs.log)

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

  gs.logItemsPushed = len(gs.log)
  gs.log = append(gs.log, fmt.Sprintf(
      "<i>%s</i> claimed <i>%s</i> spelled a word.",
      gs.players[gs.currentPlayerIdx].username, gs.lastPlayerUsername))

  isWord, err := validateWord(gs.stem)
  if err != nil {
    return err
  }
  var loser string
  var isOrIsNot string
  if isWord {
    loser = gs.lastPlayerUsername
    isOrIsNot = "IS"
    if p, ok := gs.usernameToPlayer[gs.lastPlayerUsername]; ok {
      p.incrementScore(0)
    }
  } else {
    isOrIsNot = "IS NOT"
    p := gs.players[gs.currentPlayerIdx]
    p.incrementScore(0)
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

  gs.logItemsPushed = len(gs.log)

  // make sure the challenged player hasn't left
  lastPlayerUsernameIdx := -1
  for i, p := range gs.players {
    if p.username == gs.lastPlayerUsername {
      lastPlayerUsernameIdx = i
      break
    }
  }
  if lastPlayerUsernameIdx == -1 {
    gs.log = append(gs.log, fmt.Sprintf(
        "<i>%s</i> challenged <i>%s</i>, who left the game.",
        gs.players[gs.currentPlayerIdx].username, gs.lastPlayerUsername))
    gs.newRound()
    return nil
  }
  gs.log = append(gs.log, fmt.Sprintf(
      "<i>%s</i> challenged <i>%s</i> for a continuation.",
      gs.players[gs.currentPlayerIdx].username, gs.lastPlayerUsername))

  gs.lastPlayerUsername = gs.players[gs.currentPlayerIdx].username
  if len(gs.players) == 0 {
    gs.currentPlayerIdx = 0
  } else {
    gs.currentPlayerIdx = (gs.currentPlayerIdx + 1) % len(gs.players)
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

  gs.logItemsPushed = len(gs.log)
  gs.log = append(gs.log, fmt.Sprintf("<i>%s</i> rebutted with '%s'.",
                                      gs.players[gs.currentPlayerIdx].username,
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
    loser = gs.lastPlayerUsername
    if p, ok := gs.usernameToPlayer[gs.lastPlayerUsername]; ok {
      p.incrementScore(0)
    }
  } else {
    isOrIsNot = "IS NOT"
    p := gs.players[gs.currentPlayerIdx]
    p.incrementScore(0)
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
  gs.logItemsPushed = len(gs.log)
  gs.log = append(gs.log, fmt.Sprintf(
      "<i>%s</i>: <b>%s</b>%s<b>%s</b>",
      gs.players[gs.currentPlayerIdx].username,
      strings.ToUpper(prefix), gs.stem, strings.ToUpper(suffix)))

  gs.stem = strings.ToUpper(prefix + gs.stem + suffix)

  gs.lastPlayerUsername = gs.players[gs.currentPlayerIdx].username
  if len(gs.players) == 0 {
    gs.currentPlayerIdx = 0  // Seems extremely unlikely but I'd rather be safe
  } else {
    gs.currentPlayerIdx = (gs.currentPlayerIdx + 1) % len(gs.players)
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
  p := gs.players[gs.currentPlayerIdx % len(gs.players)]
  return (p.username == cookie.Name) && (p.cookie.Value == cookie.Value)
}

func (gs *Room) newRound() {
  gs.stem = ""
  if len(gs.players) == 0 {
    gs.startingPlayerIdx = 0
  } else {
    gs.startingPlayerIdx = (gs.startingPlayerIdx + 1) % len(gs.players)
  }
  gs.lastPlayerUsername = ""
  gs.currentPlayerIdx = gs.startingPlayerIdx
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
  if index < gs.currentPlayerIdx {
    gs.currentPlayerIdx--
  }
  if index < gs.startingPlayerIdx {
    gs.startingPlayerIdx--
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

  gs.logItemsPushed = len(gs.log)

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
      if (username != gs.lastPlayerUsername &&
          username != gs.players[gs.currentPlayerIdx].username) {
        return fmt.Errorf("it is not your turn")
      }
  }
  gs.logItemsPushed = len(gs.log)

  gs.usernameToPlayer[username].incrementScore(0)
  gs.log = append(gs.log, fmt.Sprintf(
      "<i>%s</i> conceded the round. +1 <i>%s</i>", username, username))
  gs.newRound()
  return nil
}

func (gs *Room) Votekick(cookies []*http.Cookie,
                         kickRecipientUsername string) error {
  gs.mutex.Lock()
  defer gs.mutex.Unlock()

  voterUsername, ok := gs.getValidCookie(cookies)
  if !ok {
    return fmt.Errorf("no credentials provided")
  }

  kickRecipient, ok := gs.usernameToPlayer[kickRecipientUsername]
  if !ok {
    return fmt.Errorf("player not found");
  }

  err := kickRecipient.votekick(voterUsername)
  if err != nil {
    return err
  }

  gs.logItemsPushed = len(gs.log)
  gs.log = append(gs.log, fmt.Sprintf("<i>%s</i> voted to kick <i>%s</i>.",
                                      voterUsername, kickRecipientUsername))
  // if a majority has voted to kick the player, remove them from the game
  if float64(kickRecipient.numVotesToKick) >= float64(len(gs.players)) / 1.9 {
    for i, p := range gs.players {
      if p.username == kickRecipientUsername {
        gs.removePlayer(i)
        break
      }
    }
  }
  return nil
}

