package gamestate

import (
  "net/http"
  "sync"
  "strings"
  "errors"
  // "fmt"
)

type GamePhase int
const (
  kEdit GamePhase = iota
  kRebut
  kInsufficientPlayers
)

type GameState struct {
  mutex sync.RWMutex
  players []*Player
  usernameToPlayer map[string]*Player
  word string
  phase GamePhase
  nextPlayer uint
  lastPlayer uint
  firstPlayer uint
}
type JGameState struct { // For json encoding
  Players []Player  `json:"players"`
  Word string       `json:"word"`
  Phase GamePhase   `json:"phase"`
  NextPlayer uint   `json:"nextPlayer"`
  LastPlayer uint   `json:"lastPlayer"`
  FirstPlayer uint  `json:"firstPlayer"`
}

func NewGameState() *GameState {
  gs := new(GameState)
  gs.players = make([]*Player, 2)
  gs.usernameToPlayer = make(map[string]*Player)
  gs.phase = kInsufficientPlayers
  gs.nextPlayer = 0
  gs.lastPlayer = 0 // not super meaningful...
  gs.firstPlayer = 0
  return gs
}

func (gs *GameState) GetWord() string {
  gs.mutex.RLock()
  defer gs.mutex.RUnlock()
  return gs.word
}

func (gs *GameState) AffixWord(prefix string, suffix string) (string, error) {
  gs.mutex.Lock()
  defer gs.mutex.Unlock()
  gs.word = prefix + gs.word + suffix
  return gs.word, nil
}

func (gs *GameState) ValidateWord() (bool, error) {
  gs.mutex.RLock()
  defer gs.mutex.RUnlock()
  reqUri := "https://api.dictionaryapi.dev/api/v2/entries/en/" + gs.word
  resp, err := http.Get(reqUri)
  if err != nil {
    return false, err
  }
  return resp.StatusCode == http.StatusOK, nil
}

func (gs *GameState) ContainsValidCookie(cookies []*http.Cookie) bool {
  for _, cookie := range cookies {
    if gs.IsValidCookie(cookie) {
      return true
    }
  }
  return false
}

func (gs *GameState) IsValidCookie(cookie *http.Cookie) bool {
  gs.mutex.RLock()
  defer gs.mutex.RUnlock()

  if _, ok := gs.usernameToPlayer[cookie.Name]; !ok {
    return false
  }
  return gs.usernameToPlayer[cookie.Name].cookie.Value == cookie.Value
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
  gs.usernameToPlayer[username] = p
  gs.players = append(gs.players, p)
  return p.GetCookie(), nil
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
