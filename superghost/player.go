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
  lastHeartbeat time.Time
}

type JPlayer struct {
  Username string `json:"username"`
  Score uint        `json:"score"`
}

func (p *Player) MarshalJSON() ([]byte, error) {
  return json.Marshal(JPlayer {
    Username: p.username,
    Score: p.score,
  })
}

func (p *Player) heartbeat() {
  p.lastHeartbeat = time.Now()
}

func NewPlayer(username string, path string) *Player {
  p := new(Player)
  p.username = username
  p.score = 0
  p.cookie = newCookie(path, username)
  p.lastHeartbeat = time.Now()
  return p
}

