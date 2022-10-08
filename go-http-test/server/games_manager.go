package main

import (
  "encoding/base64"
  "errors"
  "math/rand"
  "net/http"
  "encoding/json"
  "time"
  "sync"
)

type GamesManager struct {
  stringToGame map[string]*Game
  games []Game
  mutex sync.RWMutex
}

// Inject player so that an update to the player creation interface does not
// necessitate a change in this interface
func (gm *GamesManager) CreateGame(playerId string) (*Game, *http.Cookie) {
  gm.mutex.Lock()
  defer gm.mutex.Unlock()

  const kGameIdLen = 8
  randomBytes := make([]byte, 32)
  var gameId string
  // Generate unique random uri
  for ok := true; ok; {
    _, err := rand.Read(randomBytes)
    if err != nil {
      panic(err)
    }
    gameId = base64.URLEncoding.EncodeToString(randomBytes)[:kGameIdLen]
    _, ok = gm.stringToGame[gameId]
  }
  // Add game
  gm.games = append(gm.games, Game{id: string(gameId),
                                   idToPlayer: make(map[string]*Player),
                                   players: make([]Player, 0)})

  gm.stringToGame[gameId] = &gm.games[len(gm.games)-1]
  player, _ := gm.stringToGame[gameId].AddPlayer(playerId)

  return gm.stringToGame[gameId], player.GetCookie()
}

func (m *GamesManager) GetGame(id string) (*Game, bool) {
  m.mutex.RLock()
  defer m.mutex.RUnlock()

  game, ok := m.stringToGame[id]
  return game, ok
}

func (m *GamesManager) GetAllGames() []Game {
  m.mutex.RLock()
  defer m.mutex.RUnlock()

  return m.games
}

type Game struct {
  id string
  players []Player
  idToPlayer map[string]*Player
  word string
  whoseTurn uint
  roundsPlayed uint
  mutex sync.RWMutex
}

type JGame struct {
  Id string
  Players []Player
  Word string
  WhoseTurn uint
  RoundsPlayed uint
}
func (g *Game) MarshalJSON() ([]byte, error) {
  g.mutex.RLock()
  defer g.mutex.RUnlock()

  return json.Marshal(JGame{Id: g.id,
                            Players: g.players,
                            Word: g.word,
                            WhoseTurn: g.whoseTurn,})
}

func (g *Game) AddPlayer(id string) (*Player, error) {
  // Validate display name
  g.mutex.Lock()
  defer g.mutex.Unlock()

  if _, ok := g.idToPlayer[id]; len(id) == 0 || ok {
    return nil, errors.New("`displayName` must be unique non-empty string.")
  }

  // Generate a cookie
  randomBytes := make([]byte, 32)
  _, err := rand.Read(randomBytes)
  if err != nil {
    panic(err)
  }
  cookieVal := base64.StdEncoding.EncodeToString(randomBytes)[:32]

  cookie := http.Cookie{
    Name: id,
    Value: cookieVal,
    Expires: time.Now().Add(24 * time.Hour),
    Path: "/api/v0/games/" + g.id,
  }
  g.players = append(g.players, Player{id: id, strikes: 0, cookie: cookie})
  g.idToPlayer[id] = &g.players[len(g.players)-1]

  return g.idToPlayer[id], nil
}

func (g *Game) GetPlayer(id string) (*Player, bool) {
  g.mutex.RLock()
  defer g.mutex.RUnlock()

  player, ok := g.idToPlayer[id]
  return player, ok
}

func (g *Game) GetAllPlayers() []Player {
  g.mutex.RLock()
  defer g.mutex.RUnlock()

  // Lock all players
  for _, p := range g.players {
    p.mutex.RLock()
    defer p.mutex.RUnlock()
  }

  return g.players
}

func (g *Game) PostPrefix(letter string) string {
  g.mutex.Lock()
  defer g.mutex.Unlock()

  g.word = letter + g.word

  g.whoseTurn += 1
  return g.word
}

func (g *Game) PostSuffix(letter string) string {
  g.mutex.Lock()
  defer g.mutex.Unlock()

  g.word = g.word + letter

  g.whoseTurn += 1
  return g.word
}

func (g *Game) RequestContainsInTurnCookie(r *http.Request) bool {
  g.mutex.RLock()
  defer g.mutex.RUnlock()

  playerIdx := (g.roundsPlayed + g.whoseTurn) % uint(len(g.players))
  player := &g.players[playerIdx]

  player.mutex.RLock()
  defer player.mutex.RUnlock()

  cookie, err := r.Cookie(player.id)
  return err == nil && cookie.Value == player.cookie.Value
}

func (g *Game) RequestContainsValidCookie(r *http.Request) (*Player, bool) {
  g.mutex.RLock()
  defer g.mutex.RUnlock()

  for _, cookie := range r.Cookies() {
    if p, ok := g.idToPlayer[cookie.Name]; ok && p.CookieMatches(cookie) {
      return p, true
    }
  }
  return nil, false
}

func (g *Game) GetWord() string {
  g.mutex.RLock()
  defer g.mutex.RUnlock()

  return g.word
}

type Player struct {
  id string
  strikes int
  cookie http.Cookie
  mutex sync.RWMutex
}

type JPlayer struct {
  Id string `json:"id"`
  Strikes int `json:"strikes"`
}
func (p *Player) MarshalJSON() ([]byte, error) {
  p.mutex.RLock()
  defer p.mutex.RUnlock()

  return json.Marshal(JPlayer{Id: p.id, Strikes: p.strikes})
}

func (p *Player) GetCookie() *http.Cookie {
  p.mutex.RLock()
  defer p.mutex.RUnlock()

  return &p.cookie
}

func (p *Player) CookieMatches(c *http.Cookie) bool {
  p.mutex.RLock()
  defer p.mutex.RUnlock()

  return c.Name == p.cookie.Name && c.Value == p.cookie.Value
}
