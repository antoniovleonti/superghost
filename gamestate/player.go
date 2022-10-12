package gamestate

import(
  "net/http"
  "math/rand"
  "encoding/base64"
  "time"
)

type Player struct {
  username string
  score uint
  cookie *http.Cookie
}

func NewPlayer(username string) *Player {
  p := new(Player)
  p.username = username
  p.score = 0
  p.cookie = NewCookie(username)
  return p
}

func NewCookie(username string) *http.Cookie {
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

func (p *Player) GetCookie() *http.Cookie {
  return p.cookie
}
