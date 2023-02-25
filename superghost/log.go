package superghost

type logItemType string
const (
  kJoin logItemType = "Join"
  kLeave logItemType = "Leave"
  kChallengeIsWord logItemType = "ChallengeIsWord"
  kChallengeContinuation logItemType = "ChallengeContinuation"
  kChallengeResult logItemType = "ChallengeResult"
  kChallengedPlayerLeft logItemType = "ChallengedPlayerLeft"
  kRebuttal logItemType = "Rebut"
  kAffix logItemType = "Affix"
  kConcede logItemType = "Concede"
  kEliminated logItemType = "Eliminated"
  kVoteToKick logItemType = "VoteToKick"
  kKick logItemType = "Kick"
  kGameOver logItemType = "GameOver"
  kGameStart logItemType = "GameStart"
  kTimeout logItemType = "Timeout"
  kInsufficientPlayers logItemType = "InsufficientPlayers"
  kReadyUp logItemType = "ReadyUp"
)

type logItem struct {
  Type logItemType
  From string `json:",omitempty"`
  To string `json:",omitempty"`
  Prefix string `json:",omitempty"`
  Suffix string `json:",omitempty"`
  Stem string `json:",omitempty"`
  Success *bool `json:",omitempty"`
}

type BufferedLog struct {
  history []logItem
  itemsPushed int  // The number of log items already sent to clients
}

func newBufferedLog() *BufferedLog {
  bl := new(BufferedLog)
  bl.history = make([]logItem, 0)
  return bl
}

func (bl *BufferedLog) flush() {
  bl.itemsPushed = len(bl.history)
}

func (bl *BufferedLog) appendJoin(username string) {
  bl.history = append(bl.history, logItem{
                        Type: kJoin,
                        From: username,
                      })
}

func (bl *BufferedLog) appendChallengeIsWord(challenger string,
                                             recipient string) {
  bl.history = append(bl.history, logItem{
                        Type: kChallengeIsWord,
                        From: challenger,
                        To: recipient,
                      })
}

func (bl *BufferedLog) appendChallengeResult(stem string, isWord bool,
                                             loser string) {
  tmp := logItem{
    Type: kChallengeResult,
    Stem: stem,
    Success: new(bool),
    To: loser,
  }
  *tmp.Success = isWord
  bl.history = append(bl.history, tmp)
}

func (bl *BufferedLog) appendChallengedPlayerLeft(challenger,
                                                  recipient string) {
  bl.history = append(bl.history, logItem{
                        Type: kChallengedPlayerLeft,
                        From: challenger,
                        To: recipient,
                      })
}

func (bl *BufferedLog) appendChallengeContinuation(challenger,
                                                   recipient string) {
  bl.history = append(bl.history, logItem{
                        Type: kChallengeContinuation,
                        From: challenger,
                        To: recipient,
                      })
}

func (bl *BufferedLog) appendRebuttal(username, stem, prefix, suffix string) {
  bl.history = append(bl.history, logItem{
                        Type: kRebuttal,
                        From: username,
                        Stem: stem,
                        Prefix: prefix,
                        Suffix: suffix,
                      })
}

func (bl *BufferedLog) appendAffixation(username, prefix, stem, suffix string) {
  bl.history = append(bl.history, logItem{
                        Type: kAffix,
                        From: username,
                        Stem: stem,
                        Prefix: prefix,
                        Suffix: suffix,
                      })
}

func (bl *BufferedLog) appendLeave(username string) {
  bl.history = append(bl.history, logItem{
                        Type: kLeave,
                        From: username,
                      })
}

func (bl *BufferedLog) appendConcession(username string) {
  bl.history = append(bl.history, logItem{
                        Type: kConcede,
                        From: username,
                      })
}

func (bl *BufferedLog) appendElimination(username string) {
  bl.history = append(bl.history, logItem{
                        Type: kEliminated,
                        From: username,
                      })
}

func (bl *BufferedLog) appendVoteToKick(voter string, recipient string) {
  bl.history = append(bl.history, logItem{
                        Type: kVoteToKick,
                        From: voter,
                        To: recipient,
                      })
}

func (bl *BufferedLog) appendKick(username string) {
  bl.history = append(bl.history, logItem{
                        Type: kKick,
                        To: username,
                      })
}

func (bl *BufferedLog) appendGameOver(username string) {
bl.history = append(bl.history, logItem{
                      Type: kGameOver,
                      To: username,
                    })
}

func (bl *BufferedLog) appendTimeout(username string) {
  bl.history = append(bl.history, logItem{
                        Type: kTimeout,
                        From: username,
                      })
}

func (bl *BufferedLog) appendInsufficientPlayers() {
  bl.history = append(bl.history, logItem{ Type: kInsufficientPlayers })
}

func (bl *BufferedLog) appendGameStart() {
  bl.history = append(bl.history, logItem{ Type: kGameStart })
}

func (bl *BufferedLog) appendReadyUp(username string) {
  bl.history = append(bl.history, logItem{
                        Type: kReadyUp,
                        From: username,
                      })
}
