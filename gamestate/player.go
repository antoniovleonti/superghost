package gamestate

import(
  "encoding/base64"
  "encoding/json"
  "math/rand"
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

func NewPlayer(username string) *Player {
  p := new(Player)
  p.username = username
  p.score = 0
  p.cookie = newCookie(username)
  p.lastHeartbeat = time.Now()
  return p
}

func newCookie(username string) *http.Cookie {
  randomBytes := make([]byte, 32)
  _, err := rand.Read(randomBytes)
  if err != nil {
    panic(err)
  }
  cookieVal := base64.StdEncoding.EncodeToString(randomBytes)[:32]

  c := new(http.Cookie)
  c.Name = username
  c.Value = cookieVal
  c.Expires = time.Now().Add(24 * time.Hour)
  c.Path = "/"
  return c
}

func init() {
  rand.Seed(time.Now().UnixNano())
}

