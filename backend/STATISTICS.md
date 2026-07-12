# Statistics system

## Storage model

Statistics use counters rather than an event row for every card action:

1. The game engine updates `PlayerGameStatistics` only after a game action succeeds.
2. Those counters are included in the persisted lobby snapshot, so a server restart does not lose an active game's progress.
3. Each started game receives a stable UUID. When it ends normally or by forfeit, the server writes one `games` row and at most one `game_player_statistics` row per authenticated player.
4. The same transaction updates each player's cached `player_statistics` row. The game UUID makes completion retries idempotent, so a retry cannot double-count totals.

Mutual endings and technical aborts do not affect ranked statistics. Guests still affect player count and placement, but only authenticated users receive profile statistics.

`game_player_statistics` is the auditable per-game source. `player_statistics` is a small lifetime cache for inexpensive profiles and leaderboards. If an aggregation rule ever changes, the cache can be rebuilt from the per-game rows.

## Short examples

If a player draws from the deck, creates a four-card run containing a joker, and discards, the successful actions add:

- `cards_drawn_from_deck += 1`
- `compositions_created += 1`
- `runs_created += 1`
- `cards_played += 4`
- `jokers_played += 1`
- `turns_taken += 1`
- `cards_discarded += 1`

If that player wins while opponents receive 18 and 27 penalty points:

- the winner gets `rounds_won += 1` and `points_inflicted += 45`
- the opponents get 18 and 27 added to `penalty_points`
- `largest_round_points_inflicted` records 45 for the winner
- cards and hand points remaining are accumulated for later averages

Most UI values are derived rather than stored redundantly:

```text
win rate                  = games_won / games_played
average placement         = total_placement / games_played
round win rate            = rounds_won / rounds_played
discard draw preference   = cards_drawn_from_discard / (deck draws + discard draws)
run share                 = runs_created / compositions_created
set share                 = sets_created / compositions_created
completion rate           = compositions_completed / compositions_created
average penalty per round = penalty_points / rounds_played
reliability               = 1 - forfeits / games_played
```

All divisions should use `NULLIF(denominator, 0)` and the UI should show an eligibility state until the sample is meaningful.

## Badges and percentile awards

The existing data supports the proposed badges without storing badge-specific action logs:

| Badge | Possible rule |
| --- | --- |
| Fast Opener | fastest opening turn, or average per-game opening record from game history |
| Joker Thief | joker reclaims per round or per game |
| Suit Collector | same-suit special wins |
| Pair Master | six-pairs special wins |
| Big Punisher | points inflicted per round or largest single-round punishment |
| Survivor | wins with high final penalty points, or wins after a large round penalty |

For a dynamic "top X%" badge, first define an eligible population and then rank a normalized metric. For example, a Joker Thief badge could require at least 10 games and rank `jokers_reclaimed / rounds_played`:

```sql
WITH eligible AS (
    SELECT
        user_id,
        jokers_reclaimed::numeric / NULLIF(rounds_played, 0) AS reclaim_rate
    FROM player_statistics
    WHERE games_played >= 10 AND rounds_played >= 25
), ranked AS (
    SELECT
        user_id,
        reclaim_rate,
        PERCENT_RANK() OVER (ORDER BY reclaim_rate DESC) AS percentile
    FROM eligible
)
SELECT user_id, reclaim_rate
FROM ranked
WHERE percentile <= 0.05;
```

Recommended safeguards:

- require minimum games/rounds so one lucky game does not dominate
- rank rates for play-style badges and totals for longevity badges
- use lower-is-better ordering for placement, penalties, or opening speed
- recalculate dynamic percentile badges periodically because the population changes
- add an absolute threshold as well as a percentile when the eligible population is small
- use season/date filters on `games.completed_at` if seasonal leaderboards are introduced

When badges are implemented, dynamic badges can be calculated on read or cached periodically. Permanent achievements should use small `badge_definitions` and `user_badges` tables; action-level rows are still unnecessary because the per-game statistics contain the evidence needed for evaluation.

## Extending the system

For a new counter:

1. Add it to `PlayerGameStatistics` and update it at the successful game-engine transition that owns the behavior.
2. Include it in `CompletedGamePlayerRecord`.
3. Add the column to both per-game and lifetime tables in a new migration.
4. Add it to the transactional insert/upsert and regenerate sqlc models.
5. Test the action counter, restart persistence, idempotent completion, and lifetime aggregation.

Prefer counters, sums, maxima, and streak boundaries. Add detailed event or per-round tables only when a concrete feature cannot be reconstructed from per-game rows; this keeps storage proportional to players per completed game rather than turns or cards played.
