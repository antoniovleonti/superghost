package superghost

import(
  "crypto/rand"
  "encoding/base64"
  "net/http"
  "regexp"
  "time"
)

var _alphanumPattern *regexp.Regexp
var _lowerPattern *regexp.Regexp

type GameConfig struct {
  maxPlayers int
  minWordLength int
}

type SuperGhostGame struct {
  State *GameState
  config *GameConfig
}

func NewSuperGhostGame(config GameConfig) *SuperGhostGame {
  gs := new(GameState)
  gs.players = make([]*Player, 0)
  gs.usernameToPlayer = make(map[string]*Player)
  gs.mode = kInsufficientPlayers
  gs.nextPlayer = 0
  gs.lastPlayer = ""
  gs.firstPlayer = 0
  // copy config
  gc := new(GameConfig)
  gc.maxPlayers = config.maxPlayers
  gc.minWordLength = gc.minWordLength

  sgg := new(SuperGhostGame)
  sgg.State = gs
  sgg.config = gc
  return sgg
}

func validateWord(word string) (isWord bool, err error) {
  reqUri := "https://api.dictionaryapi.dev/api/v2/entries/en/" + word
  resp, err := http.Get(reqUri)
  if err != nil {
    return false, err
  }
  return resp.StatusCode == http.StatusOK, nil
}

func newCookie(username string) *http.Cookie {
  c := new(http.Cookie)
  c.Name = username
  c.Value = getRandBase64String(32)
  c.Expires = time.Now().Add(24 * time.Hour)
  c.Path = "/"
  return c
}

func getRandBase64String(length int) string {
  randomBytes := make([]byte, length)
  _, err := rand.Read(randomBytes)
  if err != nil {
    panic(err)
  }
  return base64.StdEncoding.EncodeToString(randomBytes)[:length]
}

func init() {
  _alphanumPattern = regexp.MustCompile("^[[:alnum:]]+$")
  _lowerPattern = regexp.MustCompile("^[[:lower:]]+$")
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