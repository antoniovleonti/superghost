package superghost

import(
  "encoding/json"
  "net/http"
  "fmt"
  "time"
)

type Player struct {
  username string
  score uint
  cookie *http.Cookie
  numVotesToKick uint
  whoVotedToKick map[string]bool
  lastHeartbeat time.Time
  isEliminated bool
}

type JPlayer struct {
  Username string
  Score uint
  NumVotesToKick uint
  IsEliminated bool
}

func (p *Player) MarshalJSON() ([]byte, error) {
  return json.Marshal(JPlayer {
    Username: p.username,
    Score: p.score,
    NumVotesToKick: p.numVotesToKick,
    IsEliminated: p.isEliminated,
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

func (p *Player) votekick(voterUsername string) error {
  // username has already been validated
  hasAlreadyVoted, ok := p.whoVotedToKick[voterUsername]
  if !ok || !hasAlreadyVoted /* useful if votes can be revoked */ {
    p.whoVotedToKick[voterUsername] = true
    p.numVotesToKick++
  } else {
    return fmt.Errorf("you've already voted to kick this player")
  }
  return nil
}

func (p *Player) incrementScore(eliminationThreshold int) (isEliminated bool) {
  p.score++
  if eliminationThreshold > 0 && int(p.score) >= eliminationThreshold {
    p.isEliminated = true
  }
  return p.isEliminated
}

