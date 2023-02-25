"use_strict"

// Used to test the game log
function testGameLog() {
  let msgTypes = [
    "Join",
    "Leave",
    "CallengeIsWord",
    "ChallengeContinuation",
    "ChallengeResult",
    "ChallengedPlayerLeft",
    "Rebut",
    "Affix",
    "Concede",
    "Eliminated",
    "VoteToKick",
    "Kick",
    "GameOver",
    "GameStart",
    "Timeout",
    "InsufficientPlayers",
    "ReadyUp",
  ];
  for (const t of msgTypes) {
    appendToGameLog([{Type: t,
                     From: "Antonio",
                     To: "Devin",
                     Prefix: "s",
                     Suffix: "s",
                     Stem: "tupid",
                     Succcess: false,}]);
  }
}

function testPopulatePlayerList() {
  const players = [
    {
      Username: "Antonio",
      Score: 5,
      TimeRemaining: 100000,
    },
    {
      Username: "Devin",
      Score: 9,
      TimeRemaining: 100000000000,
    },
    {
      Username: "Grayson",
      Score: 3,
      TimeRemaining: 200000000000,
    },
  ];
  const currentPlayer = "";
  const deadline = new Date(); // 2 min from now;
  deadline.setMinutes(deadline.getMinutes() + 2);
  populatePlayerList(players, "edit", currentPlayer, deadline);
}

function testChat() {
  setInterval(() => appendToChat({
        Sender: "Antonio", 
        Content: "This is a test! How do I look? I should take up two lines. " +
                 "Or maybe even three!"}), 
      1000);
}

function testWriteShortStatus() {
  const states = [ "edit", "rebut", "waiting to start", "invalid" ];
  let i = 0;
  setInterval(function () {
    writeShortStatus("nextPlayer", "lastPlayer", states[i]);
    i = (i + 1) % states.length;
  }, 5000);
}

function testUpdateButtons() {
  const states = [ "edit", "rebut" ];
  const players = [ "Antonio", "Devin" ];
  const lastPlayers = [ "Devin", "Antonio" ];
  let i = 0;
  let j = 0;
  setInterval(function() {
    writeShortStatus(players[i], lastPlayers[i], states[j]);
    updateButtons({State: states[j], CurrentPlayerUsername: players[i]});
    i = (i + 1);
    if (i == players.length) {
      j = (j + 1) % states.length;
    }
    i %= players.length;
  }, 5000);
}

function testPopulateConfig() {
  const config = {
    "MaxPlayers": 2,
    "MinWordLength": 5,
    "IsPublic": true,
    "EliminationThreshold": 5,
    "AllowRepeatWords": false,
    "PlayerTimePerWord": 120000000000,
    "PauseAtRoundStart": false
  };
  populateConfig(config);
}

function testRenderGameStates(myUsername) {
	const roomStates = [
		{
			"Players": [
				{
					"Username": "Antonio",
					"Score": 0,
					"NumVotesToKick": 0,
					"IsEliminated": false,
					"IsReady": true,
					"TimeRemaining": 120000000000
				},
				{
					"Username": "Dummy",
					"Score": 0,
					"NumVotesToKick": 0,
					"IsEliminated": false,
					"IsReady": true,
					"TimeRemaining": 120000000000
				}
			],
			"Stem": "",
			"State": "edit",
			"CurrentPlayerUsername": "Antonio",
			"CurrentPlayerDeadline": "2023-02-15T21:34:46.1805947Z",
			"LastPlayerUsername": "",
			"StartingPlayerIdx": 0,
			"LogPush": [
				{
					"Type": "ReadyUp",
					"From": "Antonio"
				},
				{
					"Type": "GameStart"
				}
			]
		},
		{
			"Players": [
				{
					"Username": "Antonio",
					"Score": 0,
					"NumVotesToKick": 0,
					"IsEliminated": false,
					"IsReady": true,
					"TimeRemaining": 111013672656
				},
				{
					"Username": "Dummy",
					"Score": 0,
					"NumVotesToKick": 0,
					"IsEliminated": false,
					"IsReady": true,
					"TimeRemaining": 120000000000
				}
			],
			"Stem": "R",
			"State": "edit",
			"CurrentPlayerUsername": "Dummy",
			"CurrentPlayerDeadline": "2023-02-15T21:34:55.166924623Z",
			"LastPlayerUsername": "Antonio",
			"StartingPlayerIdx": 0,
			"LogPush": [
				{
					"Type": "Affix",
					"From": "Antonio",
					"Suffix": "R"
				}
			]
		},
		{
			"Players": [
				{
					"Username": "Antonio",
					"Score": 0,
					"NumVotesToKick": 0,
					"IsEliminated": false,
					"IsReady": true,
					"TimeRemaining": 111013672656
				},
				{
					"Username": "Dummy",
					"Score": 0,
					"NumVotesToKick": 0,
					"IsEliminated": false,
					"IsReady": true,
					"TimeRemaining": 117354510155
				}
			],
			"Stem": "UR",
			"State": "edit",
			"CurrentPlayerUsername": "Antonio",
			"CurrentPlayerDeadline": "2023-02-15T21:34:48.82608957Z",
			"LastPlayerUsername": "Dummy",
			"StartingPlayerIdx": 0,
			"LogPush": [
				{
					"Type": "Affix",
					"From": "Dummy",
					"Prefix": "U",
					"Stem": "R"
				}
			]
		},
		{
			"Players": [
				{
					"Username": "Antonio",
					"Score": 0,
					"NumVotesToKick": 0,
					"IsEliminated": false,
					"IsReady": true,
					"TimeRemaining": 108766410709
				},
				{
					"Username": "Dummy",
					"Score": 0,
					"NumVotesToKick": 0,
					"IsEliminated": false,
					"IsReady": true,
					"TimeRemaining": 117354510155
				}
			],
			"Stem": "URG",
			"State": "edit",
			"CurrentPlayerUsername": "Dummy",
			"CurrentPlayerDeadline": "2023-02-15T21:34:57.414191706Z",
			"LastPlayerUsername": "Antonio",
			"StartingPlayerIdx": 0,
			"LogPush": [
				{
					"Type": "Affix",
					"From": "Antonio",
					"Suffix": "G",
					"Stem": "UR"
				}
			]
		},
		{
			"Players": [
				{
					"Username": "Antonio",
					"Score": 0,
					"NumVotesToKick": 0,
					"IsEliminated": false,
					"IsReady": true,
					"TimeRemaining": 108766410709
				},
				{
					"Username": "Dummy",
					"Score": 0,
					"NumVotesToKick": 0,
					"IsEliminated": false,
					"IsReady": true,
					"TimeRemaining": 115223716833
				}
			],
			"Stem": "URGE",
			"State": "edit",
			"CurrentPlayerUsername": "Antonio",
			"CurrentPlayerDeadline": "2023-02-15T21:34:50.956889316Z",
			"LastPlayerUsername": "Dummy",
			"StartingPlayerIdx": 0,
			"LogPush": [
				{
					"Type": "Affix",
					"From": "Dummy",
					"Suffix": "E",
					"Stem": "URG"
				}
			]
		},
		{
			"Players": [
				{
					"Username": "Antonio",
					"Score": 0,
					"NumVotesToKick": 0,
					"IsEliminated": false,
					"IsReady": true,
					"TimeRemaining": 106362780246
				},
				{
					"Username": "Dummy",
					"Score": 0,
					"NumVotesToKick": 0,
					"IsEliminated": false,
					"IsReady": true,
					"TimeRemaining": 115223716833
				}
			],
			"Stem": "URGEN",
			"State": "edit",
			"CurrentPlayerUsername": "Dummy",
			"CurrentPlayerDeadline": "2023-02-15T21:34:59.817828675Z",
			"LastPlayerUsername": "Antonio",
			"StartingPlayerIdx": 0,
			"LogPush": [
				{
					"Type": "Affix",
					"From": "Antonio",
					"Suffix": "N",
					"Stem": "URGE"
				}
			]
		},
		{
			"Players": [
				{
					"Username": "Antonio",
					"Score": 0,
					"NumVotesToKick": 0,
					"IsEliminated": false,
					"IsReady": true,
					"TimeRemaining": 106362780246
				},
				{
					"Username": "Dummy",
					"Score": 0,
					"NumVotesToKick": 0,
					"IsEliminated": false,
					"IsReady": true,
					"TimeRemaining": 112936754285
				}
			],
			"Stem": "URGEN",
			"State": "rebut",
			"CurrentPlayerUsername": "Antonio",
			"CurrentPlayerDeadline": "2023-02-15T21:34:53.243856602Z",
			"LastPlayerUsername": "Dummy",
			"StartingPlayerIdx": 0,
			"LogPush": [
				{
					"Type": "ChallengeContinuation",
					"From": "Dummy",
					"To": "Antonio"
				}
			]
		},
		{
			"Players": [
				{
					"Username": "Antonio",
					"Score": 1,
					"NumVotesToKick": 0,
					"IsEliminated": false,
					"IsReady": true,
					"TimeRemaining": 120000000000
				},
				{
					"Username": "Dummy",
					"Score": 0,
					"NumVotesToKick": 0,
					"IsEliminated": false,
					"IsReady": true,
					"TimeRemaining": 120000000000
				}
			],
			"Stem": "",
			"State": "edit",
			"CurrentPlayerUsername": "Dummy",
			"CurrentPlayerDeadline": "2023-02-15T21:35:13.760712553Z",
			"LastPlayerUsername": "",
			"StartingPlayerIdx": 1,
			"LogPush": [
				{
					"Type": "Rebut",
					"From": "Antonio",
					"Suffix": "T",
					"Stem": "URGEN"
				},
				{
					"Type": "ChallengeResult",
					"To": "Antonio",
					"Stem": "URGENT"
				}
			]
		}
	];

  let i = 0;
  setInterval(function () {
    renderGameState(roomStates[i], myUsername);
    i = (i + 1) % roomStates.length;
  }, 500);
}

