package gamestate

import (
  "net/http"
  "sync"
  "strings"
  "errors"
  "encoding/json"
  "fmt"
)

type GamePhase int
const (
  kEdit GamePhase = iota
  kRebut
  kInsufficientPlayers
)
func (p GamePhase) String() string {
  switch p {
    case kEdit:
      return "kEdit"
    case kRebut:
      return "kRebut"
    case kInsufficientPlayers:
      return "kInsufficientPlayers"
    default:
      panic("unsupported value!")
  }
}

type GameState struct {
  mutex sync.RWMutex
  players []*Player
  usernameToPlayer map[string]*Player
  word string
  phase GamePhase
  nextPlayer uint
  lastPlayer string
  firstPlayer uint
}
type JGameState struct { // publicly visible version of gamestate
  Players []*Player  `json:"players"`
  Word string       `json:"word"`
  Phase string   `json:"phase"`
  NextPlayer uint   `json:"nextPlayer"`
  LastPlayer string `json:"lastPlayer"`
  FirstPlayer uint  `json:"firstPlayer"`
}

func (gs *GameState) MarshalJSON() ([]byte, error) {
  gs.mutex.RLock()
  defer gs.mutex.RUnlock()

  return json.Marshal(JGameState {
    Players: gs.players,
    Word: gs.word,
    Phase: gs.phase.String(),
    NextPlayer: gs.nextPlayer % uint(len(gs.players)),
    LastPlayer: gs.lastPlayer,
    FirstPlayer: gs.firstPlayer % uint(len(gs.players)),
  })
}

func NewGameState() *GameState {
  gs := new(GameState)
  gs.players = make([]*Player, 0)
  gs.usernameToPlayer = make(map[string]*Player)
  gs.phase = kInsufficientPlayers
  gs.nextPlayer = 0
  gs.lastPlayer = ""
  gs.firstPlayer = 0
  return gs
}

func (gs *GameState) AffixWord(prefix string, suffix string) (string, error) {
  gs.mutex.Lock()
  defer gs.mutex.Unlock()
  if gs.phase != kEdit {
    return "", errors.New(fmt.Sprintf("cannot edit word in %s mode",
                                       gs.phase.String()))
  }
  gs.word = prefix + gs.word + suffix
  gs.lastPlayer = gs.players[gs.nextPlayer % uint(len(gs.players))].username
  gs.nextPlayer++
  return gs.word, nil
}

func validateWord(word string) (bool, error) {
  reqUri := "https://api.dictionaryapi.dev/api/v2/entries/en/" + word
  resp, err := http.Get(reqUri)
  if err != nil {
    return false, err
  }
  return resp.StatusCode == http.StatusOK, nil
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

func (gs *GameState) GetInTurnCookie(cookies []*http.Cookie) (
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

  p := gs.players[gs.nextPlayer % uint(len(gs.players))]
  return (p.username == cookie.Name) && (p.cookie.Value == cookie.Value)
}

func (gs *GameState) AddPlayer(username string) (*http.Cookie, error) {
  gs.mutex.Lock()
  defer gs.mutex.Unlock()

  // sanitize & validate username
  username = strings.TrimSpace(username)
  if _, ok := gs.usernameToPlayer[username]; ok {
    return nil, errors.New("username already in use")
  }
  p := NewPlayer(username)

  if (len(gs.players) > 0) {
    gs.nextPlayer %= uint(len(gs.players))
    gs.firstPlayer %= uint(len(gs.players))
  }

  gs.usernameToPlayer[username] = p
  gs.players = append(gs.players, p)
  if len(gs.players) >= 2 && gs.phase == kInsufficientPlayers {
    gs.phase = kEdit
  }
  if len(gs.players) < 2 {
    gs.phase = kInsufficientPlayers
    gs.newRound()
  }
  return p.GetCookie(), nil
}

func (gs *GameState) newRound() {
  gs.word = ""
  gs.firstPlayer++
  gs.lastPlayer = ""
  gs.nextPlayer = gs.firstPlayer
  gs.phase = kEdit
}

func (gs *GameState) ChallengeIsWord() error {
  gs.mutex.Lock()
  defer gs.mutex.Unlock()

  if gs.phase != kEdit {
    return errors.New(fmt.Sprintf("cannot challenge in %s mode",
                                  gs.phase.String()))
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
    gs.players[gs.nextPlayer % uint(len(gs.players))].score++
  }
  gs.newRound()
  return nil
}

func (gs *GameState) ChallengeContinuation() error {
  gs.mutex.Lock()
  defer gs.mutex.Unlock()

  if gs.phase != kEdit {
    return errors.New(fmt.Sprintf("cannot challenge in %s mode",
                                  gs.phase.String()))
  }

  tmpNextPlayer := gs.nextPlayer
  foundLastPlayer := false
  // make sure the challenged player hasn't left
  for i, p := range gs.players {
    if p.username == gs.lastPlayer {
      foundLastPlayer = true
      gs.nextPlayer = uint(i)
    }
  }
  if !foundLastPlayer {
    gs.newRound()
  }

  gs.lastPlayer = gs.players[tmpNextPlayer % uint(len(gs.players))].username
  gs.phase = kRebut
  return nil
}

func (gs *GameState) RebutChallenge(continuation string) error {
  if gs.phase != kRebut {
    return errors.New(fmt.Sprintf("cannot rebut in %s mode", gs.phase.String()))
  }
  // clean up word
  continuation = strings.TrimSpace(continuation)
  // verify continuation contains current word
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
    gs.players[gs.nextPlayer % uint(len(gs.players))].score++
  }
  gs.newRound()
  return nil
}

// Generated by https://mholt.github.io/json-to-go/
// type Entry []struct {
// 	Word      string `json:"word"`
// 	Phonetic  string `json:"phonetic,omitempty"`
// 	Phonetics []struct {
// 		Text      string `json:"text"`
// 		Audio     string `json:"audio"`
// 		SourceURL string `json:"sourceUrl,omitempty"`
// 	} `json:"phonetics"`
// 	Meanings []struct {
// 		PartOfSpeech string `json:"partOfSpeech"`
// 		Definitions  []struct {
// 			Definition string        `json:"definition"`
// 			Synonyms   []string `json:"synonyms"`
// 			Antonyms   []string `json:"antonyms"`
// 		} `json:"definitions"`
// 		Synonyms []string `json:"synonyms"`
// 		Antonyms []string `json:"antonyms"`
// 	} `json:"meanings"`
// 	License struct {
// 		Name string `json:"name"`
// 		URL  string `json:"url"`
// 	} `json:"license"`
//	SourceUrls []string `json:"sourceUrls"`
// }
