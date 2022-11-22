package superghost

import (
  "encoding/json"
  "fmt"
  "net/http"
  "strings"
  "sync"
)

type State int
const (
  kEdit State = iota
  kRebut
  kInsufficientPlayers
  kGameOver
)
func (p State) String() string {
  switch p {
    case kEdit:
      return "edit"
    case kRebut:
      return "rebut"
    case kInsufficientPlayers:
      return "insufficient players"
    case kGameOver:
      return "game over"
    default:
      panic("invalid State value")
  }
}

type Config struct {
  MaxPlayers int `json:"maxPlayers"`
  MinStemLength int `json:"minStemLength"`
  IsPublic bool `json:"isPublic"`
  EliminationThreshold int `json:"eliminationThreshold"`
}

type Room struct {
  config *Config
  mutex sync.RWMutex

  pm *playerManager

  stem string
  state State

  log *BufferedLog
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

func (r *Room) MarshalJSON() ([]byte, error) {
  r.mutex.RLock()
  defer r.mutex.RUnlock()

  return json.Marshal(jRoom {
    Players: r.pm.players,
    Stem: strings.ToUpper(r.stem),
    State: r.state.String(),
    CurrentPlayerIdx: r.pm.currentPlayerIdx,
    LastPlayerUsername: r.pm.lastPlayerUsername,
    StartingPlayerIdx: r.pm.startingPlayerIdx,
    LogPush: r.log.history[r.log.itemsPushed:],
  })
}

func (r *Room) MarshalJSONConfig() ([]byte, error) {
  return json.Marshal(r.config)
}

func (r *Room) MarshalJSONFullLog() ([]byte, error) {
  r.mutex.RLock()
  defer r.mutex.RUnlock()

  return json.Marshal(jRoom {
    Players: r.pm.players,
    Stem: strings.ToUpper(r.stem),
    State: r.state.String(),
    CurrentPlayerIdx: r.pm.currentPlayerIdx,
    LastPlayerUsername: r.pm.lastPlayerUsername,
    StartingPlayerIdx: r.pm.startingPlayerIdx,
    LogPush: r.log.history,
  })
}

func NewRoom(config Config) *Room {
  r := new(Room)

  r.config = new(Config)
  r.config.MaxPlayers = config.MaxPlayers
  r.config.MinStemLength = config.MinStemLength
  r.config.IsPublic = config.IsPublic
  r.config.EliminationThreshold = config.EliminationThreshold

  r.pm = newPlayerManager()
  r.state = kInsufficientPlayers
  r.log = newBufferedLog()
  return r
}

// public, mutex-protected version
func (r *Room) GetValidCookie(cookies []*http.Cookie) (string, bool) {
  r.mutex.RLock()
  defer r.mutex.RUnlock()
  return r.pm.getValidCookie(cookies)
}


func (r *Room) AddPlayer(username string, path string) (*http.Cookie, error) {
  r.mutex.Lock()
  defer r.mutex.Unlock()

  if len(r.pm.players) >= r.config.MaxPlayers {
    return nil, fmt.Errorf("player limit reached")
  }
  if !_usernamePattern.MatchString(username) {
    return nil, fmt.Errorf("username must be alphanumeric")
  }
  if _, ok := r.pm.usernameToPlayer[username]; ok {
    return nil, fmt.Errorf("username '%s' already in use", username)
  }

  r.log.flush()

  p := NewPlayer(username, path)
  r.pm.usernameToPlayer[username] = p
  r.pm.players = append(r.pm.players, p)

  r.log.appendJoin(username)

  if len(r.pm.players) >= 2 && r.state == kInsufficientPlayers {
    r.state = kEdit
  }
  if len(r.pm.players) < 2 {
    r.state = kInsufficientPlayers
  }
  return p.cookie, nil
}

func (r *Room) ChallengeIsWord(cookies []*http.Cookie) error {
  r.mutex.Lock()
  defer r.mutex.Unlock()

  if _, ok := r.pm.getInTurnCookie(cookies); !ok {
    return fmt.Errorf("it is not your turn")
  }
  if r.state != kEdit {
    return fmt.Errorf("cannot challenge right now")
  }
  if len(r.stem) < r.config.MinStemLength {
    return fmt.Errorf("minimum word length not met")
  }

  r.log.flush()
  r.log.appendChallengeIsWord(r.pm.currentPlayerUsername(),
                              r.pm.lastPlayerUsername)

  isWord, err := validateWord(r.stem)
  if err != nil {
    return err
  }
  var loser string
  if isWord {
    loser = r.pm.lastPlayerUsername
    if p, ok := r.pm.usernameToPlayer[r.pm.lastPlayerUsername]; ok {
      p.incrementScore(r.config.EliminationThreshold)
    }
  } else {
    p := r.pm.players[r.pm.currentPlayerIdx]
    p.incrementScore(r.config.EliminationThreshold)
    loser = p.username
  }
  r.log.appendChallengeResult(strings.ToUpper(r.stem), isWord, loser)
  r.newRound()
  return nil
}

func (r *Room) ChallengeContinuation(cookies []*http.Cookie) error {
  r.mutex.Lock()
  defer r.mutex.Unlock()

  if _, ok := r.pm.getInTurnCookie(cookies); !ok {
    return fmt.Errorf("it is not your turn")
  }
  if r.state != kEdit {
    return fmt.Errorf("cannot challenge right now")
  }
  if len(r.stem) < 1 {
    return fmt.Errorf("cannot challenge empty stem")
  }

  r.log.flush()

  if !r.pm.swapCurrentAndLastPlayers() {
    r.log.appendChallengedPlayerLeft(r.pm.currentPlayerUsername(),
                                     r.pm.lastPlayerUsername)
    r.newRound()
    return nil
  }
  r.log.appendChallengeContinuation(r.pm.currentPlayerUsername(),
                                    r.pm.lastPlayerUsername)
  r.state = kRebut
  return nil
}

func (r *Room) RebutChallenge(cookies []*http.Cookie,
                              prefix string,
                              suffix string) error {
  r.mutex.Lock()
  defer r.mutex.Unlock()

  if r.state != kRebut {
    return fmt.Errorf("cannot rebut right now")
  }
  if _, ok := r.pm.getInTurnCookie(cookies); !ok {
    return fmt.Errorf("it is not your turn")
  }

  continuation := strings.ToUpper(prefix + r.stem + suffix)
  if len(continuation) < r.config.MinStemLength {
    return fmt.Errorf("minimum word length not met")
  }

  r.log.flush()
  r.log.appendRebuttal(r.pm.currentPlayerUsername(), continuation)
  // check if it is a word
  isWord, err := validateWord(continuation)
  if err != nil {
    return err
  }
  // update game Room accordingly
  var loser string
  if isWord {
    // challenger gets a letter
    loser = r.pm.lastPlayerUsername
    if p, ok := r.pm.usernameToPlayer[r.pm.lastPlayerUsername]; ok {
      p.incrementScore(r.config.EliminationThreshold)
    }
  } else {
    p := r.pm.players[r.pm.currentPlayerIdx]
    p.incrementScore(r.config.EliminationThreshold)
    loser = p.username
  }
  r.log.appendChallengeResult(continuation, isWord, loser)
  r.newRound()
  return nil
}

func (r *Room) AffixWord(
    cookies []*http.Cookie, prefix string, suffix string) error {
  r.mutex.Lock()
  defer r.mutex.Unlock()
  if r.state != kEdit {
    return fmt.Errorf("cannot affix right now")
  }
  if _, ok := r.pm.getInTurnCookie(cookies); !ok {
    return fmt.Errorf("it is not your turn")
  }
  if !_alphaPattern.MatchString(prefix + suffix) {
    return fmt.Errorf(
        "exactly one alphabetical prefix OR suffix must be provided " +
        "(received: {prefix: '%s', suffix: '%s'})", prefix, suffix)
  }

  // update log
  r.log.flush()
  r.log.appendAffixation(r.pm.currentPlayerUsername(), prefix, r.stem, suffix)

  r.stem = strings.ToUpper(prefix + r.stem + suffix)

  r.pm.incrementCurrentPlayer()
  return nil
}

func (r *Room) newRound() {
  r.stem = ""
  r.pm.incrementStartingPlayer()
  if ok, winner := r.pm.onlyOnePlayerRemaining(); ok {
    // Game has ended
    r.log.appendGameOver(winner)
    r.pm.resetScores()
    return
  }
  r.pm.currentPlayerIdx = r.pm.startingPlayerIdx
  if len(r.pm.players) >= 2 {
    r.state = kEdit
  } else {
    r.state = kInsufficientPlayers
  }
}

func (r *Room) Leave(cookies []*http.Cookie) error {
  r.mutex.Lock()
  defer r.mutex.Unlock()

  username, ok := r.pm.getValidCookie(cookies)
  if !ok {
    return fmt.Errorf("no credentials provided")
  }

  r.log.flush()

  if err := r.pm.removePlayer(username); err != nil {
    return err
  }
  r.log.appendLeave(username)
  return nil
}

func (r *Room) Concede(cookies []*http.Cookie) error {
  r.mutex.Lock()
  defer r.mutex.Unlock()

  username, ok := r.pm.getValidCookie(cookies)
  if !ok {
    return fmt.Errorf("no credentials provided")
  }
  switch r.state {

    case kInsufficientPlayers:
      return fmt.Errorf("cannot concede right now")

    case kEdit:
      if len(r.stem) == 0 {
        return fmt.Errorf("cannot concede when word is empty")
      }

    case kRebut:
      if (username != r.pm.lastPlayerUsername &&
          username != r.pm.players[r.pm.currentPlayerIdx].username) {
        return fmt.Errorf("it is not your turn")
      }
  }
  r.log.flush()

  isEliminated := r.pm.usernameToPlayer[username].incrementScore(
      r.config.EliminationThreshold)

  r.log.appendConcession(username)
  if isEliminated {
    r.log.appendElimination(username)
  }
  r.newRound()
  return nil
}

func (r *Room) Votekick(cookies []*http.Cookie,
                        kickRecipientUsername string) error {
  r.mutex.Lock()
  defer r.mutex.Unlock()

  voterUsername, ok := r.pm.getValidCookie(cookies)
  if !ok {
    return fmt.Errorf("no credentials provided")
  }

  kickRecipient, ok := r.pm.usernameToPlayer[kickRecipientUsername]
  if !ok {
    return fmt.Errorf("player not found");
  }

  err := kickRecipient.votekick(voterUsername)
  if err != nil {
    return err
  }

  r.log.flush()
  r.log.appendVoteToKick(voterUsername, kickRecipientUsername)
  // if a majority has voted to kick the player, remove them from the game
  if float64(kickRecipient.numVotesToKick) >= float64(len(r.pm.players)) / 1.9 {
    if err := r.pm.removePlayer(kickRecipientUsername); err != nil {
      return err
    }
    r.log.appendKick(kickRecipientUsername)
  }
  return nil
}
