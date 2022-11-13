package superghost

import(
  "net/http"
  "time"
  "encoding/json"
)

type Config struct {
  MaxPlayers int
  MinWordLength int
  IsPublic bool
}

// The main "point" of this struct is to add config data to backend requests
type Room struct {
  state *state
  config *Config
}

func NewRoom(config Config) *Room {
  gs := newState()
  // copy config
  gc := new(Config)
  gc.MaxPlayers = config.MaxPlayers
  gc.MinWordLength = config.MinWordLength
  gc.IsPublic = config.IsPublic

  r := new(Room)
  r.state = gs
  r.config = gc
  return r
}

func (r *Room) GetJsonGameState() ([]byte, error) {
  return json.Marshal(r.state)
}

func (r *Room) GetValidCookie(cookies []*http.Cookie) (string, bool) {
  return r.state.GetValidCookie(cookies)
}

func (r *Room) AddPlayer(username string, path string) (
    *http.Cookie, error) {
  return r.state.AddPlayer(username, path, r.config.MaxPlayers)
}

func (r *Room) ChallengeIsWord(cookies []*http.Cookie) error {
  return r.state.ChallengeIsWord(cookies, r.config.MinWordLength)
}

func (r *Room) ChallengeContinuation(cookies []*http.Cookie) error {
  return r.state.ChallengeContinuation(cookies)
}

func (r *Room) RebutChallenge(
    cookies []*http.Cookie, prefix string, suffix string) error {
  return r.state.RebutChallenge(cookies, prefix, suffix, r.config.MinWordLength)
}

func (r *Room) AffixWord(
    cookies []*http.Cookie, prefix string, suffix string) error {
  return r.state.AffixWord(cookies, prefix, suffix)
}

func (r *Room) Heartbeat(cookies []*http.Cookie) error {
  return r.state.Heartbeat(cookies)
}

func (r *Room) RemoveDeadPlayers(duration time.Duration) bool {
  return r.state.RemoveDeadPlayers(duration)
}

func (r *Room) Concede(cookies []*http.Cookie) error {
  return r.state.Concede(cookies)
}

func (r *Room) Leave(cookies []*http.Cookie) error {
  return r.state.Leave(cookies)
}

func (r *Room) Votekick(cookies []*http.Cookie, usernameToKick string) error {
  return r.state.Votekick(cookies, usernameToKick)
}

