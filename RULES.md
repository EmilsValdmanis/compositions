# Compositions – Game Rules

## Overview

Kompozīcijas is a turn-based card game played with **2 standard decks** (including jokers).  
The goal is to **get rid of all cards** by forming valid compositions and minimize points across rounds.

The game is played over multiple rounds until a player wins the overall game.

---

## Setup

- Use **2 shuffled decks** (including jokers)
- Each player is dealt **12 cards**
- Remaining cards form the **draw deck**
- One card is placed as the starting **discard pile**
---

## Objective

- Primary goal: **be the first to get rid of all of your cards**
- Long-term goal: **force all other players to exceed 100 points**

---

## Turn Structure

Each turn consists of:

1. **Draw a card**
   - From the deck or discard pile if you can use in that turn for a composition or adding to a composition

2. **Play compositions or add a card to an existing one (optional)**
   - Only allowed if initial requirement is met (see below)

3. **Discard one card**
   - Ends the turn

---

## Initial Requirement (40 Points Rule)

- To place your first compositions the total points you place on board must be **at least 40 points**
- Until this is done, you **cannot play any cards to the table**

---

## Compositions

A composition is a valid set of cards placed on the table.

### Types of Compositions

#### 1. Set (Same Rank)
- 3 or more cards
- Same rank (e.g. three 7s)
- Different suits

#### 2. Run (Sequence)
- 3 or more cards
- Sequential ranks (e.g. 5-6-7-8)
- Same suit

---

## Jokers

- Jokers can represent **any card**
- Can be used in both sets and runs or added to an existing composition
- If left in hand at end of round → **20 points**

### Aces
- An Ace can act as:
  - **High (like a face card)** → counts as 10 points
  - **Low (value of 1)** → used in sequences like Ace-2-3

- The value of the Ace depends on how it is used in the composition:

#### Examples
- Three Aces (different suits):
  - 10 + 10 + 10 = **30 points**

- Run: Ace, Two, Three:
  - 1 + 2 + 3 = **6 points**

- If an Ace remains in hand at the end of a round:
  - Typically counts as **10 points** (unless it was the final card played, depending on rules)

---

### Notes
- Ace behavior must be determined **contextually**:
  - In sets → usually treated as high (10)
  - In low runs (Ace-2-3) → treated as 1
- Validation logic must account for both interpretations

---

## Card Values (Scoring)

| Card Type      | Points |
|----------------|--------|
| Number cards   | Face value (2–10) |
| Face cards     | 10     |
| Ace            | 1 or 10 (see below) |
| Joker          | 20     |

### Ace Special Rule
- Counts as **1 point if it is the last card**
- Otherwise counts as **10 points**

---

## Ending a Round

A round ends when:
- A player **plays all their cards**

---

## Scoring After Round

All other players:
- Count points of remaining cards in hand
- Add those points to their **total score**

---

## Winning the Game

- The game continues across multiple rounds
- Players accumulate points

### Elimination / Adjustment Rule
- When a player exceeds **100 points**, they are at risk
- If **not all players exceed 100 points**:
  - Players above 100 reset to the **highest score among remaining players**

### Final Winner
- The player who forces **all other players above 100 points** wins

---

## Special Winning Conditions

### 1. Same Suit Collection
- A player collects all **12 cards of the same suit**
- Immediate win on turn

### 2. Six Pairs
- A player forms **6 identical pairs**
- Counts as a special win condition

---

## Notes & Edge Cases

- The game uses **2 decks**, so duplicate cards exist
- Jokers can substitute missing cards in compositions
- Composition validation must handle:
  - Joker substitution
  - Gaps in sequences
  - Duplicate cards across decks

---

## Summary

- Draw → Play (if allowed) → Discard
- First valid play must be ≥ 40 points
- Use compositions to reduce hand
- End round by emptying hand
- Accumulate points → avoid exceeding 100
- Last player under threshold wins
