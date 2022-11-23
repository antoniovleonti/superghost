package superghost

import (
  "fmt"
  "net/http"
)

type playerManager struct {
  players []*Player
  usernameToPlayer map[string]*Player

  currentPlayerIdx int

  lastPlayerUsername string
  startingPlayerIdx int
}

func newPlayerManager() *playerManager {
  pm := new(playerManager)
  pm.players = make([]*Player, 0)
  pm.usernameToPlayer = make(map[string]*Player)
  return pm
}

func (pm *playerManager) currentPlayerUsername() string {
  return pm.players[pm.currentPlayerIdx].username
}

func (pm *playerManager) addPlayer(username string, path string) (*http.Cookie, error) {
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

func (pm *playerManager) removePlayer(username string) error {
  for i, p := range pm.players {
    if p.username == username {
      return pm.removePlayerByIdx(i)
      break
    }
  }
  return fmt.Errorf("player not found")
}

func (pm *playerManager) removePlayerByIdx(index int) error {
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

func (pm *playerManager) getValidCookie(cookies []*http.Cookie) (string, bool) {
  for _, cookie := range cookies {
    if pm.isValidCookie(cookie) {
      return cookie.Name, true
    }
  }
  return "", false
}

func (pm *playerManager) getInTurnCookie(
    cookies []*http.Cookie) (*http.Cookie, bool) {
  for _, cookie := range cookies {
    if pm.isInTurnCookie(cookie) {
      return cookie, true
    }
  }
  return nil, false
}

func (pm *playerManager) isInTurnCookie(cookie *http.Cookie) bool {
  p := pm.players[pm.currentPlayerIdx % len(pm.players)]
  return (p.username == cookie.Name) && (p.cookie.Value == cookie.Value)
}

func (pm *playerManager) isValidCookie(cookie *http.Cookie) bool {

  player, ok := pm.usernameToPlayer[cookie.Name]
  if !ok {
    return false
  }
  return player.cookie.Value == cookie.Value
}

func (pm *playerManager) incrementCurrentPlayer() (ok bool) {
  if len(pm.players) == 0 {
    pm.currentPlayerIdx = 0  // Seems extremely unlikely but I'd rather be safe
    return false
  }
  for i := (pm.currentPlayerIdx + 1) % len(pm.players);
      i != pm.currentPlayerIdx;
      i = (i + 1) % len(pm.players) {
    if !pm.players[i].isEliminated {
      pm.lastPlayerUsername = pm.players[pm.currentPlayerIdx].username
      pm.currentPlayerIdx = i
      return true
    }
  }
  // Couldn't find a valid player (strange)
  return false
}

func (pm *playerManager) incrementStartingPlayer() (ok bool) {
  if len(pm.players) == 0 {
    return false
  }
  for i := (pm.startingPlayerIdx + 1) % len(pm.players);
      i != pm.startingPlayerIdx;
      i = (i + 1) % len(pm.players) {
    if !pm.players[i].isEliminated {
      pm.lastPlayerUsername = ""
      pm.startingPlayerIdx = i
      return true
    }
  }
  // Couldn't find a valid player (strange)
  return false
}

func (pm *playerManager) swapCurrentAndLastPlayers() (ok bool) {
  for i, p := range pm.players {
    if p.username == pm.lastPlayerUsername {
      pm.lastPlayerUsername = pm.currentPlayerUsername()
      pm.currentPlayerIdx = i
      return true
    }
  }
  return false // couldn't find last player
}

func (pm *playerManager) resetScores() {
  for _, p := range pm.players {
    p.score = 0
  }
}

func (pm *playerManager) onlyOnePlayerRemaining() (bool, string) {
  nRemaining := 0
  var winner string
  for _, p := range pm.players {
    if !p.isEliminated {
      nRemaining++
      if nRemaining > 1 {
        return false, ""
      }
      winner = p.username
    }
  }
  return true, winner
}
