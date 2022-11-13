package superghost

import(
  "encoding/json"
  "net/http"
  "time"
)

type Player struct {
  username string
  score uint
  cookie *http.Cookie
  numVotesToKick uint
  whoVotedToKick map[string]bool
  lastHeartbeat time.Time
}

type JPlayer struct {
  Username string `json:"username"`
  Score uint        `json:"score"`
  NumVotesToKick uint `json:"numVotesToKick"`
}

func (p *Player) MarshalJSON() ([]byte, error) {
  return json.Marshal(JPlayer {
    Username: p.username,
    Score: p.score,
    NumVotesToKick: p.numVotesToKick,
  })
}

func (p *Player) heartbeat() {
  p.lastHeartbeat = time.Now()
}

func NewPlayer(username string, path string) *Player {
  p := new(Player)
  p.username = username
  p.cookie = newCookie(path, username)
  p.lastHeartbeat = time.Now()
  p.whoVotedToKick = make(map[string]bool, 0)
  return p
}

func (p *Player) votekick(voterUsername string) {
  // username has already been validated
  hasAlreadyVoted, ok := p.whoVotedToKick[voterUsername]
  if !ok || !hasAlreadyVoted /* useful if votes can be revoked */ {
    p.whoVotedToKick[voterUsername] = true
    p.numVotesToKick++
  }
}

