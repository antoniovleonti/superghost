package superghost

import(
  "encoding/json"
  "net/http"
  "time"
)

type Player struct {
  username string
  cookie *http.Cookie

  score uint
  isEliminated bool

  // Not a countdown timer-- only accurate when it is not this player's turn
  timeRemaining time.Duration
}

type JPlayer struct {
  Username string
  Score uint
  IsEliminated bool
  TimeRemaining time.Duration
}

func (p *Player) MarshalJSON() ([]byte, error) {
  return json.Marshal(JPlayer {
    Username: p.username,
    Score: p.score,
    IsEliminated: p.isEliminated,
    TimeRemaining: p.timeRemaining,
  })
}

func NewPlayer(username string, path string,
               startingTime time.Duration) *Player {
  p := new(Player)
  p.username = username
  p.cookie = newCookie(path, username)
  p.timeRemaining = startingTime

  return p
}

func (p *Player) incrementScore(eliminationThreshold int) (isEliminated bool) {
  p.score++
  if eliminationThreshold > 0 && int(p.score) >= eliminationThreshold {
    p.isEliminated = true
  }
  return p.isEliminated
}

