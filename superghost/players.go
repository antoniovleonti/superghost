package superghost

import (
  "fmt"
  "net/http"
)

type PlayersManager struct {
  players []*Player
  usernameToPlayer map[string]*Player

  currentPlayerIdx int
  currentPlayerUsername string

  lastPlayerUsername string
  startingPlayerIdx int
}

func newPlayersManager() *PlayersManager {
  pm := new(PlayersManager)
  pm.players = make([]*Player, 0)
  pm.usernameToPlayer = make(map[string]*Player)
  return pm
}

func (pm *PlayersManager) AddPlayer(username string, path string) (*http.Cookie, error) {
  if !_usernamePattern.MatchString(username) {
    return nil, fmt.Errorf("username must be alphanumeric")
  }
  if _, ok := pm.usernameToPlayer[username]; ok {
    return nil, fmt.Errorf("username '%s' already in use", username)
  }

  p := NewPlayer(username, path)
  pm.usernameToPlayer[username] = p

  return p.cookie, nil
}

func (pm *PlayersManager) removePlayer(username string) error {
  for i, p := range pm.players {
    if p.username == username {
      return pm.removePlayerByIdx(i)
      break
    }
  }
  return fmt.Errorf("player not found")
}

func (pm *PlayersManager) removePlayerByIdx(index int) error {
  if index > len(pm.players) {
    return fmt.Errorf("index out of bounds")
  }
  if index < pm.currentPlayerIdx {
    pm.currentPlayerIdx--
  }
  if index < pm.startingPlayerIdx {
    pm.startingPlayerIdx--
  }

  delete(pm.usernameToPlayer, pm.players[index].username)

  if (index == len(pm.players) - 1) {
    pm.players = pm.players[:index] // avoid out of bounds...
  } else {
    pm.players = append(pm.players[:index], pm.players[index+1:]...)
  }
  return nil
}

func (pm *PlayersManager) getValidCookie(cookies []*http.Cookie) (string, bool) {
  for _, cookie := range cookies {
    if pm.isValidCookie(cookie) {
      return cookie.Name, true
    }
  }
  return "", false
}

func (pm *PlayersManager) getInTurnCookie(
    cookies []*http.Cookie) (*http.Cookie, bool) {
  for _, cookie := range cookies {
    if pm.isInTurnCookie(cookie) {
      return cookie, true
    }
  }
  return nil, false
}

func (pm *PlayersManager) isInTurnCookie(cookie *http.Cookie) bool {
  p := pm.players[pm.currentPlayerIdx % len(pm.players)]
  return (p.username == cookie.Name) && (p.cookie.Value == cookie.Value)
}

func (pm *PlayersManager) isValidCookie(cookie *http.Cookie) bool {

  player, ok := pm.usernameToPlayer[cookie.Name]
  if !ok {
    return false
  }
  return player.cookie.Value == cookie.Value
}
