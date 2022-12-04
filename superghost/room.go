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
  kWaitingToStart
)
func (p State) String() string {
  switch p {
    case kEdit:
      return "edit"
    case kRebut:
      return "rebut"
    case kWaitingToStart:
      return "waiting to start"
    default:
      panic("invalid State value")
  }
}

type Config struct {
  MaxPlayers int
  MinWordLength int
  IsPublic bool
  EliminationThreshold int
  AllowRepeatWords bool
  PlayerTimePerWord time.Duration
}

type Message struct {
  Sender string
  Content string
}

type Room struct {
  config *Config

  pm *playerManager

  stem string
  state State
  usedWords map[string]bool

  log *BufferedLog

  mutex sync.RWMutex

  endTurnCh chan bool
  asyncUpdateCh chan<- bool

  lastTouch time.Time
}

type JRoom struct { // publicly visible version of gamestate
  Players []*Player
  Stem string
  State string
  CurrentPlayerUsername string
  CurrentPlayerDeadline time.Time
  LastPlayerUsername string
  StartingPlayerIdx int
  LogPush []string
}

func (r *Room) MarshalJSON() ([]byte, error) {
  r.mutex.RLock()
  defer r.mutex.RUnlock()

  return json.Marshal(JRoom {
    Players: r.pm.players,
    Stem: strings.ToUpper(r.stem),
    State: r.state.String(),
    CurrentPlayerUsername: r.pm.currentPlayerUsername(),
    CurrentPlayerDeadline: r.pm.currentPlayerDeadline,
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

  return json.Marshal(JRoom {
    Players: r.pm.players,
    Stem: strings.ToUpper(r.stem),
    State: r.state.String(),
    CurrentPlayerUsername: r.pm.currentPlayerUsername(),
    CurrentPlayerDeadline: r.pm.currentPlayerDeadline,
    LastPlayerUsername: r.pm.lastPlayerUsername,
    StartingPlayerIdx: r.pm.startingPlayerIdx,
    LogPush: r.log.history,
  })
}

type JRoomMetadata struct {
  PlayerCount int
  MaxPlayers int
  EliminationThreshold int
  MinWordLength int
  ID string
}

func NewRoom(config Config, asyncUpdateCh chan<- bool) *Room {
  r := new(Room)

  r.config = new(Config)
  r.config.MaxPlayers = config.MaxPlayers
  r.config.MinWordLength = config.MinWordLength
  r.config.IsPublic = config.IsPublic
  r.config.EliminationThreshold = config.EliminationThreshold
  r.config.PlayerTimePerWord = config.PlayerTimePerWord

  r.asyncUpdateCh = asyncUpdateCh
  r.endTurnCh = make(chan bool)

  r.pm = newPlayerManager()
  r.waitToStart()
  r.usedWords = make(map[string]bool)
  r.log = newBufferedLog()
  return r
}

func (r *Room) Metadata(ID string) JRoomMetadata {
  r.mutex.RLock()
  defer r.mutex.RUnlock()

  return JRoomMetadata {
    PlayerCount: len(r.pm.players),
    MaxPlayers: r.config.MaxPlayers,
    EliminationThreshold: r.config.EliminationThreshold,
    MinWordLength: r.config.MinWordLength,
    ID: ID,
  }
}

func (r *Room) IsPublic() bool {
  return r.config.IsPublic
}

func (r *Room) LastTouch() time.Time {
  return r.lastTouch
}

func (r *Room) updateLastTouch() {
  r.lastTouch = time.Now()
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

  r.updateLastTouch()

  if len(r.pm.players) >= r.config.MaxPlayers {
    return nil, fmt.Errorf("player limit reached")
  }

  cookie, err := r.pm.addPlayer(username, path, r.config.PlayerTimePerWord)
  if err != nil {
    return nil, err
  }

  r.log.flush()
  r.log.appendJoin(username)

  return cookie, nil
}

func (r *Room) ChallengeIsWord(cookies []*http.Cookie) error {
  r.mutex.Lock()
  defer r.mutex.Unlock()

  r.updateLastTouch()

  if _, ok := r.pm.getInTurnCookie(cookies); !ok {
    return fmt.Errorf("it is not your turn")
  }
  if r.state != kEdit {
    return fmt.Errorf("cannot challenge right now")
  }
  if len(r.stem) < r.config.MinWordLength {
    return fmt.Errorf("minimum word length not met")
  }

  r.log.flush()
  r.log.appendChallengeIsWord(r.pm.currentPlayerUsername(),
                              r.pm.lastPlayerUsername)

  // Even if the player's time expires here, we have the mutex, so it won't be
  // acted on until after we validate the word. If the validation errors,
  // however, the player is SOL
  isWord, err := validateWord(r.stem, r.usedWords, r.config.AllowRepeatWords)
  if err != nil {
    return err
  }

  r.endTurn()

  var loser string
  if isWord {
    r.usedWords[r.stem] = true
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
  r.newRoundOrGame()
  return nil
}

func (r *Room) ChallengeContinuation(cookies []*http.Cookie) error {
  r.mutex.Lock()
  defer r.mutex.Unlock()

  r.updateLastTouch()

  if _, ok := r.pm.getInTurnCookie(cookies); !ok {
    return fmt.Errorf("it is not your turn")
  }
  if r.state != kEdit {
    return fmt.Errorf("cannot challenge right now")
  }
  if len(r.stem) < 1 {
    return fmt.Errorf("cannot challenge empty stem")
  }

  r.endTurn()

  r.log.flush()

  if !r.pm.swapCurrentAndLastPlayers() {
    r.log.appendChallengedPlayerLeft(r.pm.currentPlayerUsername(),
                                     r.pm.lastPlayerUsername)
    r.newRoundOrGame()
    return nil
  }
  r.startTurnAndCountdown(r.pm.currentPlayerUsername())
  r.log.appendChallengeContinuation(r.pm.lastPlayerUsername,
                                    r.pm.currentPlayerUsername())
  r.state = kRebut
  return nil
}

func (r *Room) RebutChallenge(cookies []*http.Cookie,
                              prefix string,
                              suffix string) error {
  r.mutex.Lock()
  defer r.mutex.Unlock()

  r.updateLastTouch()

  if r.state != kRebut {
    return fmt.Errorf("cannot rebut right now")
  }
  if _, ok := r.pm.getInTurnCookie(cookies); !ok {
    return fmt.Errorf("it is not your turn")
  }

  continuation := strings.ToUpper(prefix + r.stem + suffix)
  if len(continuation) < r.config.MinWordLength {
    return fmt.Errorf("minimum word length not met")
  }

  r.log.flush()
  r.log.appendRebuttal(r.pm.currentPlayerUsername(), continuation)
  // check if it is a word
  isWord, err := validateWord(continuation, r.usedWords,
                              r.config.AllowRepeatWords)
  if err != nil {
    return err
  }

  r.endTurn()

  // update game Room accordingly
  var loser string
  if isWord {
    // challenger gets a letter
    r.usedWords[continuation] = true
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
  r.newRoundOrGame()
  return nil
}

func (r *Room) AffixWord(
    cookies []*http.Cookie, prefix string, suffix string) error {
  r.mutex.Lock()
  defer r.mutex.Unlock()

  r.updateLastTouch()

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

  r.endTurn()

  // update log
  r.log.flush()
  r.log.appendAffixation(r.pm.currentPlayerUsername(), prefix, r.stem, suffix)

  r.stem = strings.ToUpper(prefix + r.stem + suffix)

  r.pm.incrementCurrentPlayer()
  r.startTurnAndCountdown(r.pm.currentPlayerUsername())

  return nil
}

func (r *Room) newRoundOrGame() {
  r.stem = ""
  r.state = kEdit

  if weHaveAWinner, winner := r.pm.onlyOnePlayerNotEliminated();
      r.pm.numReadyPlayers() < 2 || weHaveAWinner {
    if weHaveAWinner {
      r.log.appendGameOver(winner)
    } else {
      r.log.appendInsufficientPlayers()
    }
    r.pm.resetScores()
    r.pm.resetReadiness()
    r.waitToStart()
  }
  r.pm.incrementStartingPlayer()
  r.pm.currentPlayerIdx = r.pm.startingPlayerIdx
  r.pm.resetPlayerTimes(r.config.PlayerTimePerWord)

  if r.state == kEdit {
    r.startTurnAndCountdown(r.pm.currentPlayerUsername())
  }
}

func (r *Room) Leave(cookies []*http.Cookie) error {
  r.mutex.Lock()
  defer r.mutex.Unlock()

  r.updateLastTouch()

  username, ok := r.pm.getValidCookie(cookies)
  if !ok {
    return fmt.Errorf("no credentials provided")
  }

  if err := r.removePlayer(username); err != nil {
    return err
  }
  if username == r.pm.currentPlayerUsername() {
    r.endTurn()
  }
  r.log.flush()
  r.log.appendLeave(username)
  return nil
}

func (r *Room) Concede(cookies []*http.Cookie) error {
  r.mutex.Lock()
  defer r.mutex.Unlock()

  r.updateLastTouch()

  username, ok := r.pm.getValidCookie(cookies)
  if !ok {
    return fmt.Errorf("no credentials provided")
  }
  switch r.state {

    case kWaitingToStart:
      return fmt.Errorf("cannot concede right now")

    case kEdit:
      if len(r.stem) == 0 {
        return fmt.Errorf("cannot concede when word is empty")
      }
      if r.pm.usernameToPlayer[username].isEliminated {
        return fmt.Errorf("cannot concede when eliminated")
      }

    case kRebut:
      if (username != r.pm.lastPlayerUsername &&
          username != r.pm.currentPlayerUsername()) {
        return fmt.Errorf("it is not your turn")
      }
  }

  r.endTurn()

  isEliminated := r.pm.usernameToPlayer[username].incrementScore(
      r.config.EliminationThreshold)

  r.log.flush()
  r.log.appendConcession(username)
  if isEliminated {
    r.log.appendElimination(username)
  }

  r.newRoundOrGame()
  return nil
}

func (r *Room) Votekick(cookies []*http.Cookie,
                        kickRecipientUsername string) error {
  r.mutex.Lock()
  defer r.mutex.Unlock()

  r.updateLastTouch()

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
    if err := r.removePlayer(kickRecipientUsername); err != nil {
      return err
    }
    r.log.appendKick(kickRecipientUsername)
  }
  return nil
}

// Just validate request & return a message object so that the server can
// broadcast it.
func (r *Room) Chat(cookies []*http.Cookie, content string) (*Message, error) {
  r.mutex.Lock()
  r.updateLastTouch()
  r.mutex.Unlock()

  username, ok := r.GetValidCookie(cookies)
  if !ok {
    return nil, fmt.Errorf("no credentials provided")
  }
  if len(content) == 0 {
    return nil, fmt.Errorf("empty message")
  }

  msg := new(Message)
  msg.Sender = username
  msg.Content = content
  return msg, nil
}

func (r *Room) startTurnAndCountdown(expectedPlayerUsername string) {
  // Outside the go func() {} section, this is a synchronous function that runs
  // only when called by another mutex-protected function (therefor DO NOT grab
  // the mutex outside the go func() part!)
  if r.config.PlayerTimePerWord == 0 * time.Second {
    return
  }
  timesUp := time.NewTimer(time.Until(r.pm.setCurrentPlayerDeadline()))

  go func() {

    select {
      case <-timesUp.C:
        // The player has run out of time
        r.mutex.Lock()
        defer r.mutex.Unlock()

        r.updateLastTouch()

        if expectedPlayerUsername != r.pm.currentPlayerUsername() {
          // The player sent their response at the moment they ran out of time
          // but before we got the mutex lock. For now I'll just give it to
          // them but I can see this being very unintuitive if the time is
          // exactly 00:00:00s or something. This can be fixed by making sure
          // the time remaining is non-zero before accepting their input.

          // Nothing to clean up since the timer fired, just exit the function
          return
        }

        isEliminated := r.pm.currentPlayer().incrementScore(
            r.config.EliminationThreshold)

        r.log.flush()
        r.log.appendTimeout(r.pm.currentPlayerUsername())
        if isEliminated {
          r.log.appendElimination(r.pm.currentPlayerUsername())
        }

        r.newRoundOrGame()
        r.asyncUpdateCh<-true // notify the frontend of the update to game state

      case <-r.endTurnCh:
        // The player beat the clock (and currently has control over the mutex).
        // Just stop the timer and let the synchronous code take care of the
        // rest.
        timesUp.Stop()
    }
  }()
}

func (r *Room) endTurn() {
  if r.config.PlayerTimePerWord > 0 {
    r.endTurnCh<-true // This will stop the countdown thread
    r.pm.currentPlayer().timeRemaining = time.Until(r.pm.currentPlayerDeadline)
  }
}

func (r *Room) removePlayer(username string) error {
  err := r.pm.removePlayer(username)
  if err != nil {
    return err
  }
  if r.pm.numReadyPlayers() < 2 {
    r.newRoundOrGame()
  }
  return nil
}

func (r *Room) ReadyUp(cookies []*http.Cookie) error {
  r.mutex.Lock()
  defer r.mutex.Unlock()

  // check cookies
  username, ok := r.pm.getValidCookie(cookies)
  if !ok {
    return fmt.Errorf("no credentials provided")
  }
  if r.pm.usernameToPlayer[username].isReady == true {
    return fmt.Errorf("you're already ready")
  }
  // ready up
  r.pm.usernameToPlayer[username].isReady = true

  r.log.flush()
  r.log.appendReadyUp(username)
  // if all players are ready, start game (if not started already)
  if r.state == kWaitingToStart &&
      len(r.pm.players) >= 2 &&
      r.pm.allPlayersReady() {
    r.state = kEdit
    r.log.appendGameStart()
    r.startTurnAndCountdown(r.pm.currentPlayerUsername())
  }
  return nil
}

func (r *Room) waitToStart() {
  r.state = kWaitingToStart
  r.pm.currentPlayerDeadline = time.Unix(0, 0) // zero time
}
