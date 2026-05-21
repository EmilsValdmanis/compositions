# Tasks

Short implementation checklist for finishing the game logic.

- [x] 1. Add core game state types: players, round, turn, table compositions, draw pile, discard pile, and total scores. See [Turn Structure](RULES.md#turn-structure) and [Winning the Game](RULES.md#winning-the-game).
- [x] 2. Implement round setup and dealing: 2 decks, 12 cards each, draw deck creation, and starting discard card. See [Setup](RULES.md#setup) and [Shuffling, Cutting, and Dealing](RULES.md#shuffling-cutting-and-dealing).
- [x] 3. Implement the cut-or-tap choice and alternate dealing order rules. See [Shuffling, Cutting, and Dealing](RULES.md#shuffling-cutting-and-dealing).
- [x] 4. Implement turn flow: draw, optional play, final discard, and move to next player. See [Turn Structure](RULES.md#turn-structure).
- [x] 5. Implement set validation: same rank, 3+ cards, different suits, and duplicate-card handling from the double deck. See [Types of Compositions](RULES.md#types-of-compositions) and [Notes & Edge Cases](RULES.md#notes--edge-cases).
- [x] 6. Implement run validation: same suit, 3+ cards, sequential ranks, and duplicate-card handling. See [Types of Compositions](RULES.md#types-of-compositions) and [Notes & Edge Cases](RULES.md#notes--edge-cases).
- [x] 7. Implement joker usage in sets and runs, including tracking what each joker represents. See [Jokers](RULES.md#jokers).
- [x] 8. Implement ace handling in compositions so Ace can be high or low depending on context. See [Aces](RULES.md#aces) and [Ace Special Rule](RULES.md#ace-special-rule).
- [x] 9. Implement the 40-point first-play rule so players cannot place cards until they meet it. See [Initial Requirement (40 Points Rule)](RULES.md#initial-requirement-40-points-rule).
- [x] 10. Implement adding cards to existing table compositions after the first-play requirement is met. See [Turn Structure](RULES.md#turn-structure).
- [x] 11. Implement joker reclaiming by replacing it with the exact represented card, including the ambiguous-set restriction. See [Jokers](RULES.md#jokers).
- [x] 12. Implement completed composition detection and removal to the discard pile before the final discard. See [Completed Compositions](RULES.md#completed-compositions).
- [x] 13. Implement round end when a player plays all cards. See [Ending a Round](RULES.md#ending-a-round).
- [x] 14. Implement draw deck exhaustion by recycling the discard pile into a new draw deck. See [Draw Deck Exhaustion](RULES.md#draw-deck-exhaustion).
- [x] 15. Implement end-of-round scoring and the over-100 adjustment rule. See [Scoring After Round](RULES.md#scoring-after-round) and [Winning the Game](RULES.md#winning-the-game).
- [x] 16. Implement special win-condition checks: 12 cards of one suit and 6 identical pairs. See [Special Winning Conditions](RULES.md#special-winning-conditions).

Next engine and testing checklist after the core rules engine.

- [x] 1. Implement round reset and round restart flow so a finished round can cleanly produce the next round state without rebuilding `GameState` by hand.
- [x] 2. Decide and encode table/dealer progression rules between rounds, including who deals next and who acts first in the next round.
- [x] 3. Add higher-level integration tests that play through full turns and full rounds instead of only package-level unit tests.
- [x] 4. Add multi-round tests that prove scoring, over-100 adjustments, and game-over behavior across consecutive rounds.
- [x] 5. Add deterministic test helpers or fixtures for deck construction so scenario tests are easier to read than long inline card lists.
- [x] 6. Review performance of discard-pile legality search and add benchmarks around `canTakeDiscardNow` before wiring it into interactive WebSocket play.

Realtime game server checklist.

- [x] 1. Expose a minimal WebSocket lobby surface in `backend/cmd/server`, so clients can connect, create a room, join a room, reconnect, and start a game over a persistent connection.
- [x] 2. Define connection, player-session, and game-room lifecycle rules for connect, websocket-close disconnect handling, reconnect, host ownership, and room start flow.
- [x] 3. Add server-level tests for the lobby/session/start flow, including reconnect handling, invalid lobby operations, and room-state broadcasts.
- [ ] 4. Add WebSocket rate limiting for connection attempts, room creation/join attempts, and per-connection message throughput to reduce brute-force room joins and flooding.
- [x] 5. Extend the WebSocket surface so clients can submit turn actions such as draw, play, add, reclaim, and discard.
- [x] 6. Broadcast in-game state changes and action results to all room participants during active play.
- [x] 7. Add server-level integration tests that drive realistic multiplayer turn flow, including invalid actions and state broadcasts during active rounds.
- [x] 8. Add useful logging and error handling around the WebSocket lobby and game play surfaces to make debugging easier during frontend integration and testing.

Frontend starter checklist.

- [x] 1. Choose or scaffold the frontend app entrypoint that will connect to the backend WebSocket server.
- [x] 2. Define the frontend WebSocket client types for connection, room, and game messages.
- [x] 3. Build a basic lobby screen that can connect, create a room, join a room, and start a game.
- [x] 4. Add a room view that shows room code, players, seats, connected status, and host controls.
- [x] 5. Add basic lobby system UI (join, leave, copy code, share link, etc.)
- [x] 6. Add a basic in-game screen shell for hand, table compositions, draw pile, discard pile, and turn indicator.
- [x] 7. Wire frontend state updates from server room-state and game-state events.
- [ ] 8. Add frontend tests for session restore, lobby flow, and room-state rendering.
- [ ] 9. Replace the temporary dealing-choice lobby UI with the full chooser flow for cut size, tap dealing order, and non-host status messaging.
- [x] 10. Add composition assembling UI for sets and runs, including joker assignment and validation feedback (including placement on table).
- [ ] 11. Add UI for composition additions so players can drag cards onto existing table compositions after opening.
