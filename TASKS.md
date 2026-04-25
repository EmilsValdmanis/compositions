# Tasks

Short implementation checklist for finishing the game logic.

- [X] 1. Add core game state types: players, round, turn, table compositions, draw pile, discard pile, and total scores. See [Turn Structure](RULES.md#turn-structure) and [Winning the Game](RULES.md#winning-the-game).
- [X] 2. Implement round setup and dealing: 2 decks, 12 cards each, draw deck creation, and starting discard card. See [Setup](RULES.md#setup) and [Shuffling, Cutting, and Dealing](RULES.md#shuffling-cutting-and-dealing).
- [X] 3. Implement the cut-or-tap choice and alternate dealing order rules. See [Shuffling, Cutting, and Dealing](RULES.md#shuffling-cutting-and-dealing).
- [X] 4. Implement turn flow: draw, optional play, final discard, and move to next player. See [Turn Structure](RULES.md#turn-structure).
- [X] 5. Implement set validation: same rank, 3+ cards, different suits, and duplicate-card handling from the double deck. See [Types of Compositions](RULES.md#types-of-compositions) and [Notes & Edge Cases](RULES.md#notes--edge-cases).
- [X] 6. Implement run validation: same suit, 3+ cards, sequential ranks, and duplicate-card handling. See [Types of Compositions](RULES.md#types-of-compositions) and [Notes & Edge Cases](RULES.md#notes--edge-cases).
- [ ] 7. Implement joker usage in sets and runs, including tracking what each joker represents. See [Jokers](RULES.md#jokers).
- [X] 8. Implement ace handling in compositions so Ace can be high or low depending on context. See [Aces](RULES.md#aces) and [Ace Special Rule](RULES.md#ace-special-rule).
- [ ] 9. Implement the 40-point first-play rule so players cannot place cards until they meet it. See [Initial Requirement (40 Points Rule)](RULES.md#initial-requirement-40-points-rule).
- [ ] 10. Implement adding cards to existing table compositions after the first-play requirement is met. See [Turn Structure](RULES.md#turn-structure).
- [ ] 11. Implement joker reclaiming by replacing it with the exact represented card, including the ambiguous-set restriction. See [Jokers](RULES.md#jokers).
- [ ] 12. Implement completed composition detection and removal to the discard pile before the final discard. See [Completed Compositions](RULES.md#completed-compositions).
- [ ] 13. Implement round end when a player plays all cards. See [Ending a Round](RULES.md#ending-a-round).
- [ ] 14. Implement draw deck exhaustion by recycling the discard pile into a new draw deck. See [Draw Deck Exhaustion](RULES.md#draw-deck-exhaustion).
- [ ] 15. Implement end-of-round scoring and the over-100 adjustment rule. See [Scoring After Round](RULES.md#scoring-after-round) and [Winning the Game](RULES.md#winning-the-game).
- [ ] 16. Implement special win-condition checks: 12 cards of one suit and 6 identical pairs. See [Special Winning Conditions](RULES.md#special-winning-conditions).
