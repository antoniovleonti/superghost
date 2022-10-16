package superghost

import(
  "net/http"
  "time"
  "encoding/json"
)

type GameConfig struct {
  MaxPlayers int
  MinWordLength int
  IsPublic bool
}

type SuperGhostGame struct {
  state *GameState
  config *GameConfig
}

func NewSuperGhostGame(config GameConfig) *SuperGhostGame {
  gs := NewGameState()
  // copy config
  gc := new(GameConfig)
  gc.MaxPlayers = config.MaxPlayers
  gc.MinWordLength = config.MinWordLength
  gc.IsPublic = config.IsPublic

  sgg := new(SuperGhostGame)
  sgg.state = gs
  sgg.config = gc
  return sgg
}

func (sgg *SuperGhostGame) GetJsonGameState() ([]byte, error) {
  return json.Marshal(sgg.state)
}

func (sgg *SuperGhostGame) GetValidCookie(cookies []*http.Cookie) (string,
                                                                   bool) {
  return sgg.state.getValidCookie(cookies)
}

func (sgg *SuperGhostGame) AddPlayer(username string) (*http.Cookie, error) {
  return sgg.state.addPlayer(username, sgg.config.MaxPlayers)
}

func (sgg *SuperGhostGame) ChallengeIsWord(cookies []*http.Cookie) error {
  return sgg.state.challengeIsWord(cookies, sgg.config.MinWordLength)
}

func (sgg *SuperGhostGame) ChallengeContinuation(cookies []*http.Cookie) error {
  return sgg.state.challengeContinuation(cookies)
}

func (sgg *SuperGhostGame) RebutChallenge(
    cookies []*http.Cookie, continuation string) error {
  return sgg.state.rebutChallenge(cookies, continuation,
                                  sgg.config.MinWordLength)
}

func (sgg *SuperGhostGame) AffixWord(
    cookies []*http.Cookie, prefix string, suffix string) error {
  return sgg.state.affixWord(cookies, prefix, suffix)
}

func (sgg *SuperGhostGame) Heartbeat(cookies []*http.Cookie) error {
  return sgg.state.heartbeat(cookies)
}

func (sgg *SuperGhostGame) RemoveDeadPlayers(duration time.Duration) bool {
  return sgg.state.removeDeadPlayers(duration)
}
