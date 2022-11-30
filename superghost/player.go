package superghost

import(
  "encoding/json"
  "net/http"
  "fmt"
  "time"
)

type Player struct {
  username string
  cookie *http.Cookie

  score uint
  isEliminated bool

  numVotesToKick uint
  whoVotedToKick map[string]bool

  // Not a countdown timer-- only accurate when it is not this player's turn
  timeRemaining time.Duration
}

type JPlayer struct {
  Username string
  Score uint
  NumVotesToKick uint
  IsEliminated bool
  TimeRemaining time.Duration
}

func (p *Player) MarshalJSON() ([]byte, error) {
  return json.Marshal(JPlayer {
    Username: p.username,
    Score: p.score,
    NumVotesToKick: p.numVotesToKick,
    IsEliminated: p.isEliminated,
    TimeRemaining: p.timeRemaining,
  })
}

func NewPlayer(username string, path string,
               startingTime time.Duration) *Player {
  p := new(Player)
  p.username = username
  p.cookie = newCookie(path, username)
  p.whoVotedToKick = make(map[string]bool, 0)
  p.timeRemaining = startingTime
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

