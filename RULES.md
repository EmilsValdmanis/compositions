# Compositions – Game Rules

## Overview

Kompozīcijas is a turn-based card game played with **2 standard decks**
(including jokers).
The goal is to **get rid of all cards** by forming valid compositions and
minimizing points across rounds.

The game is played over multiple rounds until a player wins the overall game.

---

## Setup

- Use **2 shuffled decks** (including jokers).
- Each player is dealt **12 cards** during the dealing phase.
- After all hands are dealt, the **draw deck** is formed from the undealt
  cards, together with any set-aside packet if one was created.
- After the draw deck is formed, **one card is placed face up** to start the
  discard pile.

### Shuffling, Cutting, and Dealing

- After shuffling, the player immediately before the dealer in **clockwise
  order** normally removes any number of cards from the top of the deck and
  sets them aside face down.
- Instead of making that cut, the same player may **tap the deck** and choose
  the dealing order for the players.
- Dealing always starts from the **next player clockwise** from the dealer.
- In the normal case, cards are dealt **one at a time in round-robin order
  clockwise** until each player has 12 cards.
- If the deck was tapped, the previous player chooses which player receives
  all 12 cards first, then which player receives the next 12, and so on until
  all players have their hands.
- Once every player has 12 cards, any undealt cards remaining in the main
  dealing stack are placed **on top of** the set-aside packet; together they
  become the draw deck.
- If no packet was set aside because the deck was tapped instead, the undealt
  cards simply become the draw deck.
- After the draw deck is formed, **one card** is turned face up to begin the
  discard pile.

---

## Objective

- Primary goal: **be the first to get rid of all of your cards**.
- Long-term goal: **force all other players to exceed 100 points**.

---

## Turn Structure

Each turn consists of:

1. **Draw a card**

   - From the deck, or from the top of the discard pile if that card can be
     used immediately in that turn.
   - If you have not yet met the **40 points** initial requirement, taking the
     discard card is only allowed if it helps form compositions worth at least
     40 points in that same turn.

2. **Play compositions or add a card to an existing one** (optional)

   - Only allowed if the initial requirement is met.

3. **Discard one card**

   - Before discarding, remove any composition completed during that turn and
     place it into the discard pile.
   - Ends the turn.

---

## Initial Requirement (40 Points Rule)

- To place your first compositions, the total points you place on the board
  must be **at least 40 points**.
- Until this is done, you **cannot play any cards to the table**.

---

## Compositions

A composition is a valid set of cards placed on the table.

### Types of Compositions

#### 1. Set (Same Rank)

- 3 or more cards.
- Same rank, for example three 7s.
- Different suits.

#### 2. Run (Sequence)

- 3 or more cards.
- Sequential ranks, for example 5-6-7-8.
- Same suit.
- Because the game uses 2 decks, a same-suit run may use one Ace as low and
  the other Ace as high in the same composition.
- This means the longest possible same-suit run is a **14-card** complete run:
  Ace-2-3-4-5-6-7-8-9-10-J-Q-K-Ace.

### Completed Compositions

- A completed composition is removed from the table instead of staying in
  play.
- This removal happens **before** the current player places their final
  discard to end the turn.
- A composition counts as complete when it contains every required card for
  that pattern, including:
  - all four Ace suits in an Ace set
  - a full same-suit run covering every rank in that suit, from Ace low
    through Ace high when both Ace copies are used
- When removed, the completed composition is placed into the **discard pile**,
  and then the player places their normal discard on top to finish the turn.

---

## Jokers

- Jokers can represent **any card**.
- Jokers can be used in both sets and runs or added to an existing
  composition.
- A player may reclaim a joker from a composition on the table by replacing it
  with the **exact card** the joker is currently representing.
- After reclaiming it, that player may use the joker again during the same
  turn.
- In same-rank sets, a joker cannot be reclaimed until its value is narrowed
  to **one specific missing card**.
- If it could still represent multiple missing suits, the joker stays in
  place.
- If left in hand at end of round, a joker is worth **20 points**.

### Aces

- An Ace can act as:
  - **High (like a face card)** and count as 10 points.
  - **Low (value of 1)** in sequences such as Ace-2-3.
- The value of the Ace depends on how it is used in the composition.

#### Examples

- Three Aces (different suits):
  - 10 + 10 + 10 = **30 points**
- Run: Ace, Two, Three:
  - 1 + 2 + 3 = **6 points**
- If an Ace remains in hand at the end of a round:
  - It typically counts as **10 points**, unless it was the final card played.

### Notes

- Ace behavior must be determined **contextually**:
  - In sets, it is usually treated as high (10).
  - In low runs such as Ace-2-3, it is treated as 1.
  - In a complete same-suit run, one Ace may be low and the second Ace of that
    suit may be high.
- Validation logic must account for both interpretations.

---

## Card Values (Scoring)

| Card Type    | Points                |
| :----------- | :-------------------- |
| Number cards | Face value (2-10)     |
| Face cards   | 10                    |
| Ace          | 1 or 10 (see below)   |
| Joker        | 20                    |

### Ace Special Rule

- Counts as **1 point if it is the last card in hand**.
- Otherwise counts as **10 points**.

---

## Ending a Round

A round ends when:

- A player **plays all their cards**.

## Draw Deck Exhaustion

- If the game has not ended and the draw deck runs out, flip the entire
  discard pile over to create a **new draw deck**.
- Players then continue drawing from that new deck as normal.

---

## Scoring After Round

All other players:

- Count points of remaining cards in hand.
- Add those points to their **total score**.

---

## Winning the Game

- The game continues across multiple rounds.
- Players accumulate points.

### Elimination / Adjustment Rule

- When a player exceeds **100 points**, they are at risk.
- If **not all players exceed 100 points**:
  - Players above 100 reset to the **highest score among remaining players**.

### Final Winner

- The player who forces **all other players above 100 points** wins.

---

## Special Winning Conditions

### 1. Same Suit Collection

- A player collects all **12 cards of the same suit**.
- Immediate win on turn.

### 2. Six Pairs

- A player forms **6 identical pairs**.
- Counts as a special win condition.

---

## Notes & Edge Cases

- The game uses **2 decks**, so duplicate cards exist.
- Jokers can substitute missing cards in compositions.
- Composition validation must handle:
  - joker substitution
  - gaps in sequences
  - duplicate cards across decks

---

## Summary

- Draw → Play (if allowed) → Discard.
- First valid play must be at least 40 points.
- Use compositions to reduce hand.
- End round by emptying hand.
- Accumulate points and avoid exceeding 100.
- Last player under the threshold wins.
