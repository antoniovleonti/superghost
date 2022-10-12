package gamestate

import (
  "net/http"
  "sync"
)

type GamePhase int
const (
  kEdit GamePhase = iota
  kRebut
  kInsufficientPlayers
)

type Player struct {
  id string
  score uint
  cookie http.Cookie
}

type GameState struct {
  mutex sync.RWMutex
  players []Player
  word string
  phase GamePhase
  nextPlayer uint
  lastPlayer uint
  firstPlayer uint
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

// Generated by https://mholt.github.io/json-to-go/
type Entry []struct {
	Word      string `json:"word"`
	Phonetic  string `json:"phonetic,omitempty"`
	Phonetics []struct {
		Text      string `json:"text"`
		Audio     string `json:"audio"`
		SourceURL string `json:"sourceUrl,omitempty"`
	} `json:"phonetics"`
	Meanings []struct {
		PartOfSpeech string `json:"partOfSpeech"`
		Definitions  []struct {
			Definition string        `json:"definition"`
			Synonyms   []string `json:"synonyms"`
			Antonyms   []string `json:"antonyms"`
		} `json:"definitions"`
		Synonyms []string `json:"synonyms"`
		Antonyms []string `json:"antonyms"`
	} `json:"meanings"`
	License struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"license"`
	SourceUrls []string `json:"sourceUrls"`
}