// HTML elements
const mainPanel = document.getElementById("main-panel");
const shortStatus = document.getElementById("short-status");
const stemSpans = document.getElementsByClassName("active-stem");
const logOL = document.getElementById("game-log-list");
const chatOL = document.getElementById("chat-list");
const playersOL = document.getElementById("players-list");
const configDialog = document.getElementById("config-dialog");
const configSpan = document.getElementById("config-span");
const joinDialog = document.getElementById("join-dialog");
const joinErr = document.getElementById("join-err");

// Static buttons / forms
const affixForm = document.getElementById("affix-form");
const rebutForm = document.getElementById("rebut-form");
const joinForm = document.getElementById("join-form");
const concedeButton = document.getElementById("concede-button");
const challengeContButton = document.getElementById("ch-cont-button");
const challengeIsWordButton = document.getElementById("ch-word-button");
const showConfigButton = document.getElementById("show-config-button");
const chatForm = document.getElementById("chat-form");
const chatText = document.getElementById("chat-textarea");
const onlyEnabledOnMyTurn =
    document.getElementsByClassName("only-enabled-on-my-turn");

