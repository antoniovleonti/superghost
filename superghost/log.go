package superghost

import (
  "fmt"
  "strings"
)

type BufferedLog struct {
  history []string
  itemsPushed int  // The number of log items already sent to clients
}

func newBufferedLog() *BufferedLog {
  bl := new(BufferedLog)
  bl.history = make([]string, 0)
  return bl
}

func (bl *BufferedLog) flush() {
  bl.itemsPushed = len(bl.history)
}

func (bl *BufferedLog) appendJoin(username string) {
  bl.history = append(bl.history,
                      fmt.Sprintf("<i>%s</i> joined the game!", username))
}

func (bl *BufferedLog) appendChallengeIsWord(challenger string,
                                             recipient string) {
  bl.history = append(bl.history, fmt.Sprintf(
      "<i>%s</i> claimed <i>%s</i> spelled a word.", challenger, recipient))
}

func (bl *BufferedLog) appendChallengeResult(stem string, isWord bool,
                                             loser string) {
  isOrIsNot := "is"
  if !isWord {
    isOrIsNot += " NOT"
  }
  bl.history = append(bl.history, fmt.Sprintf("'%s' %s a word! +1 <i>%s</i>.",
                                    stem, isOrIsNot, loser))
}

func (bl *BufferedLog) appendChallengedPlayerLeft(challenger string,
                                                  recipient string) {
  bl.history = append(bl.history, fmt.Sprintf(
      "<i>%s</i> challenged <i>%s</i>, who left the game.",
      challenger, recipient))
}

func (bl *BufferedLog) appendChallengeContinuation(challenger string,
                                                   recipient string) {
  bl.history = append(bl.history, fmt.Sprintf(
      "<i>%s</i> challenged <i>%s</i> for a continuation.",
      challenger, recipient))
}

func (bl *BufferedLog) appendRebuttal(username string, continuation string) {
  bl.history = append(bl.history, fmt.Sprintf("<i>%s</i> rebutted with '%s'.",
                                              username, continuation))
}

func (bl *BufferedLog) appendAffixation(
    username string, prefix string, stem string, suffix string) {
  bl.history = append(bl.history, fmt.Sprintf("<i>%s</i>: <b>%s</b>%s<b>%s</b>",
                                              username, strings.ToUpper(prefix),
                                              stem, strings.ToUpper(suffix)))
}

func (bl *BufferedLog) appendLeave(username string) {
  bl.history = append(bl.history, fmt.Sprintf("<i>%s</i> left the game.",
                                              username))
}

func (bl *BufferedLog) appendConcession(username string) {
  bl.history = append(bl.history, fmt.Sprintf(
      "<i>%s</i> conceded the round. +1 <i>%s</i>", username, username))
}

func (bl *BufferedLog) appendElimination(username string) {
  bl.history = append(bl.history, fmt.Sprintf("<i>%s</i> has been eliminated!",
                                              username))
}

func (bl *BufferedLog) appendVoteToKick(voter string, recipient string) {
  bl.history = append(bl.history, fmt.Sprintf(
      "<i>%s</i> voted to kick <i>%s</i>.", voter, recipient))
}

func (bl *BufferedLog) appendKick(username string) {
  bl.history = append(bl.history, fmt.Sprintf(
      "<i>%s</i> was kicked from the game.", username))
}
