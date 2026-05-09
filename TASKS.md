# Tasks

Short implementation checklist for finishing the game logic.

- [X] 1. Add core game state types: players, round, turn, table compositions, draw pile, discard pile, and total scores. See [Turn Structure](RULES.md#turn-structure) and [Winning the Game](RULES.md#winning-the-game).
- [X] 2. Implement round setup and dealing: 2 decks, 12 cards each, draw deck creation, and starting discard card. See [Setup](RULES.md#setup) and [Shuffling, Cutting, and Dealing](RULES.md#shuffling-cutting-and-dealing).
- [X] 3. Implement the cut-or-tap choice and alternate dealing order rules. See [Shuffling, Cutting, and Dealing](RULES.md#shuffling-cutting-and-dealing).
- [X] 4. Implement turn flow: draw, optional play, final discard, and move to next player. See [Turn Structure](RULES.md#turn-structure).
- [X] 5. Implement set validation: same rank, 3+ cards, different suits, and duplicate-card handling from the double deck. See [Types of Compositions](RULES.md#types-of-compositions) and [Notes & Edge Cases](RULES.md#notes--edge-cases).
- [X] 6. Implement run validation: same suit, 3+ cards, sequential ranks, and duplicate-card handling. See [Types of Compositions](RULES.md#types-of-compositions) and [Notes & Edge Cases](RULES.md#notes--edge-cases).
- [X] 7. Implement joker usage in sets and runs, including tracking what each joker represents. See [Jokers](RULES.md#jokers).
- [X] 8. Implement ace handling in compositions so Ace can be high or low depending on context. See [Aces](RULES.md#aces) and [Ace Special Rule](RULES.md#ace-special-rule).
- [X] 9. Implement the 40-point first-play rule so players cannot place cards until they meet it. See [Initial Requirement (40 Points Rule)](RULES.md#initial-requirement-40-points-rule).
- [X] 10. Implement adding cards to existing table compositions after the first-play requirement is met. See [Turn Structure](RULES.md#turn-structure).
- [X] 11. Implement joker reclaiming by replacing it with the exact represented card, including the ambiguous-set restriction. See [Jokers](RULES.md#jokers).
- [X] 12. Implement completed composition detection and removal to the discard pile before the final discard. See [Completed Compositions](RULES.md#completed-compositions).
- [X] 13. Implement round end when a player plays all cards. See [Ending a Round](RULES.md#ending-a-round).
- [X] 14. Implement draw deck exhaustion by recycling the discard pile into a new draw deck. See [Draw Deck Exhaustion](RULES.md#draw-deck-exhaustion).
- [X] 15. Implement end-of-round scoring and the over-100 adjustment rule. See [Scoring After Round](RULES.md#scoring-after-round) and [Winning the Game](RULES.md#winning-the-game).
- [X] 16. Implement special win-condition checks: 12 cards of one suit and 6 identical pairs. See [Special Winning Conditions](RULES.md#special-winning-conditions).

Next engine and testing checklist after the core rules engine.

- [X] 1. Implement round reset and round restart flow so a finished round can cleanly produce the next round state without rebuilding `GameState` by hand.
- [X] 2. Decide and encode table/dealer progression rules between rounds, including who deals next and who acts first in the next round.
- [X] 3. Add higher-level integration tests that play through full turns and full rounds instead of only package-level unit tests.
- [X] 4. Add multi-round tests that prove scoring, over-100 adjustments, and game-over behavior across consecutive rounds.
- [X] 5. Add deterministic test helpers or fixtures for deck construction so scenario tests are easier to read than long inline card lists.
- [X] 6. Review performance of discard-pile legality search and add benchmarks around `canTakeDiscardNow` before wiring it into interactive WebSocket play.

Realtime game server checklist.

- [X] 1. Expose a minimal WebSocket lobby surface in `backend/cmd/server`, so clients can connect, create a room, join a room, reconnect, and start a game over a persistent connection.
- [X] 2. Define connection, player-session, and game-room lifecycle rules for connect, websocket-close disconnect handling, reconnect, host ownership, and room start flow.
- [X] 3. Add server-level tests for the lobby/session/start flow, including reconnect handling, invalid lobby operations, and room-state broadcasts.
- [ ] 4. Extend the WebSocket surface so clients can submit turn actions such as draw, play, add, reclaim, and discard.
- [ ] 5. Broadcast in-game state changes and action results to all room participants during active play.
- [ ] 6. Add server-level integration tests that drive realistic multiplayer turn flow, including invalid actions and state broadcasts during active rounds.

Frontend starter checklist.

- [X] 1. Choose or scaffold the frontend app entrypoint that will connect to the backend WebSocket server.
- [ ] 2. Define the frontend WebSocket client types for connection, room, and game messages.
- [ ] 3. Persist the client `sessionId` locally so reconnect can restore the same player session.
- [ ] 4. Build a basic lobby screen that can connect, create a room, join a room, and start a game.
- [ ] 5. Add a room view that shows room code, players, seats, connected status, and host controls.
- [ ] 6. Add a basic in-game screen shell for hand, table compositions, draw pile, discard pile, and turn indicator.
- [ ] 7. Wire frontend state updates from server room-state and game-state events.
- [ ] 8. Add frontend tests for session restore, lobby flow, and room-state rendering.
