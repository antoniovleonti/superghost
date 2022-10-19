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
  s *state
  c *Config
}

func NewRoom(config Config) *Room {
  gs := newState()
  // copy config
  gc := new(Config)
  gc.MaxPlayers = config.MaxPlayers
  gc.MinWordLength = config.MinWordLength
  gc.IsPublic = config.IsPublic

  sgg := new(Room)
  sgg.s = gs
  sgg.c = gc
  return sgg
}

func (sgg *Room) GetJsonGameState() ([]byte, error) {
  return json.Marshal(sgg.s)
}

func (sgg *Room) GetValidCookie(cookies []*http.Cookie) (string, bool) {
  return sgg.s.getValidCookie(cookies)
}

func (sgg *Room) AddPlayer(username string) (*http.Cookie, error) {
  return sgg.s.addPlayer(username, sgg.c.MaxPlayers)
}

func (sgg *Room) ChallengeIsWord(cookies []*http.Cookie) error {
  return sgg.s.challengeIsWord(cookies, sgg.c.MinWordLength)
}

func (sgg *Room) ChallengeContinuation(cookies []*http.Cookie) error {
  return sgg.s.challengeContinuation(cookies)
}

func (sgg *Room) RebutChallenge(
    cookies []*http.Cookie, continuation string, giveUp bool) error {
  return sgg.s.rebutChallenge(cookies, continuation,
                              giveUp, sgg.c.MinWordLength)
}

func (sgg *Room) AffixWord(
    cookies []*http.Cookie, prefix string, suffix string) error {
  return sgg.s.affixWord(cookies, prefix, suffix)
}

func (sgg *Room) Heartbeat(cookies []*http.Cookie) error {
  return sgg.s.heartbeat(cookies)
}

func (sgg *Room) RemoveDeadPlayers(duration time.Duration) bool {
  return sgg.s.removeDeadPlayers(duration)
}


func (sgg *Room) Concede(cookies []*http.Cookie) error {
  return sgg.s.concede(cookies)
}
