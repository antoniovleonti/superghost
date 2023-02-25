"use strict";

async function staySyncedWithChat() {
	try {
    const message = await getNextChat();
		appendToChat(message);
	} catch (err) {
    console.error(err);
  }
  await staySyncedWithChat();
}

async function staySyncedWithGame(config, myUsername) {

  try {
    const gameState = await getNextGameState();
    console.log(gameState);
    renderGameState(gameState, config, myUsername);
  } catch (err) {
    console.error(err);
  }
  await staySyncedWithGame(config, myUsername);
}

async function enterGameLoop() {
  const config = await getConfig();
  console.log(config);
  populateConfig(config);

  // The only case where cancel-leave response is not ok is when the server
  // doesn't recognize the player
  const cancelLeaveResponse = await cancelLeave();
  const hasJoined = cancelLeaveResponse.ok;
  const myUsername = hasJoined ? getUsernameFromCookie() : null;

  // Sync with the server
  getCurrentGameState().then(room => renderGameState(room, config, myUsername));
  // Stay in sync with the server
  staySyncedWithChat();
  staySyncedWithGame(config, myUsername);
}

enterGameLoop();

