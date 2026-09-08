# Statistics system

## Storage model

Statistics use counters rather than an event row for every card action:

1. The game engine updates `PlayerGameStatistics` only after a game action succeeds.
2. Those counters are included in the persisted lobby snapshot, so a server restart does not lose an active game's progress.
3. Each started game receives a stable UUID and creates an `in_progress` checkpoint.
4. The same cumulative player row is replaced after every completed round and immediately after a forfeit. There is no row per action or per round.
5. A completed game finalizes those rows and updates each player's scoped cached `player_statistics` row in the same transaction. The transition away from `in_progress` makes retries idempotent.

The lobby snapshot is written before each derived statistics checkpoint. After
the checkpoint succeeds, the server writes the snapshot again with its dirty
marker cleared. A failure before the checkpoint therefore cannot put
statistics ahead of recoverable game state; a failure after it leaves a dirty
marker that safely retries the idempotent checkpoint after restart.

Mutual endings, technical aborts, and abandoned games retain their latest per-game activity with null outcomes, but do not affect ranked lifetime statistics. Guests still affect player count and placement, but only authenticated users receive profile statistics.

`game_player_statistics` is the auditable per-game source. `player_statistics` is a lifetime cache keyed by `(user_id, game_mode, ranked)` for inexpensive profiles and leaderboards. If an aggregation rule ever changes, the cache can be rebuilt from per-game rows filtered to `games.status IN ('completed', 'forfeit')` and the desired mode/ranked scope.

Current scopes are:

- `full + ranked`: the existing multi-round game, used by ranked profiles and leaderboards
- `quick + unranked`: a single completed round, shown separately on profiles and excluded from leaderboards

Game mode and ranked eligibility are separate columns so a casual full-game or ranked quick-game queue can be introduced later without changing the storage model.

Checkpoint writes happen only at meaningful boundaries:

- game start
- completed round
- player forfeit
- final completed or outcome-free ending

This means a long turn does not generate extra database rows or writes, while completed-round and reliability information survives a later abort.

Game statuses are `in_progress`, `completed`, `forfeit`, `mutual_end`, `technical_abort`, and `abandoned`. Status describes lifecycle/outcome rather than ranked eligibility. Placement and win fields remain null for mutual endings, technical aborts, and abandoned games. Profile statistics come from the matching lifetime scope; reliability or diagnostic views that intentionally include aborted games can query the retained per-game rows directly.

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
total playtime             = sum(games.active_playtime_seconds)
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

Active playtime runs only while a game is outside the lobby/game-over phases
and every non-forfeited player is connected. It pauses on disconnect, resumes
when all active players reconnect, and is checkpointed with the lobby state so
overnight pauses and server downtime are excluded. A gap between persisted game
activities contributes at most 15 minutes, so a browser left connected while
everyone is away also cannot inflate the statistic.

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

Prefer counters, sums, maxima, and streak boundaries. Add detailed event or per-round tables only when a concrete feature cannot be reconstructed from per-game rows; this keeps storage proportional to players per game rather than turns, rounds, or cards played.

Do not add leaderboard indexes speculatively. The cached lifetime table is one
row per user and is cheap to scan while the project is small. Add a targeted
index only after a real leaderboard query and `EXPLAIN ANALYZE` show that it is
needed; dynamic ratios generally will not benefit from a simple single-column
index anyway.

## Operational constraint

The current lobby is a single in-memory writer backed by one persisted lobby
snapshot. Run one active game-server replica against a database. Supporting
multiple replicas later requires room ownership (or advisory locking) and
routing each room's WebSocket connections to its owner; statistics writes are
already transactional and idempotent, but the lobby itself is not distributed.

## Elo rating

Migration 16 adds a fresh rating ladder; historical results are not replayed.
Every player starts at 1,000. Existing leaderboard participants without a rating
row display that baseline. New players appear after their first ranked full game.

Only completed or forfeited **ranked full** games with records for all two to four
participants affect Elo. Quick, unranked, abandoned, mutually ended, technically
aborted, and incomplete-participant games do not. Checkpoints never change Elo.
Placements come from the game engine: lower is better, equal placements draw,
and forfeits finish below remaining players in the engine's forfeit order.

For each player and opponent, expected score is
`1 / (1 + 10^((opponentRating - playerRating) / 400))`.
Actual score is 1 for finishing ahead, 0.5 for tying, and 0 for finishing behind.
The change is `round(32 * sum(actual - expected) / (playerCount - 1))`, calculated
from all pre-game ratings simultaneously. Round deltas half away from zero and
floor final ratings at zero. Changes are bounded by 32 per game regardless of
player count; equal-rated duels change by 16. Rounding and the floor can cause
small rating drift. There are no placements, decay, seasons, or promotion series.

| Tier | Elo |
| --- | --- |
| Bronze | 0–1,199 |
| Silver | 1,200–1,399 |
| Gold | 1,400–1,599 |
| Platinum | 1,600–1,799 |
| Diamond | 1,800–1,999 |
| Master | 2,000–2,199 |
| Grand Master | 2,200+ |

`player_ratings` stores the current rating and rated-game count;
`game_rating_changes` records before/after values for each participant. Rating
updates and lifetime statistics commit in the same transaction as game completion.
Retries of finalized games do nothing. Participants are locked in canonical UUID
order, and calculations run after all rating rows are locked. Overlapping games
are applied in database serialization order, not client timestamps.

The leaderboard's first/default metric is `elo` in both API and UI. Existing
friends/global scopes, UUID tie-breaking, pagination and pinned placement apply.
Other metrics remain available. The API supplies tier identifiers only for Elo;
the UI translates them into English or Latvian.

Validation: `go test -race ./...`, `go test -race -tags=integration
./internal/database/...`, and `go test ./internal/rating -fuzz=FuzzCalculate
-fuzztime=10s` from `backend`; `vp check` and `vp test` from `frontend`.
Integration tests use disposable PostgreSQL containers.

### Updating Elo

`internal/rating/rules.go` owns the active calculation version, initial rating,
K factor, expectation scale, and tier thresholds. `Current()` returns a copy;
each completed game uses one copy for both calculation and audit persistence.
The first rating insert and leaderboard fallback both use its initial rating.
Migration 17 removes the database's duplicate starting-rating default and adds
`game_rating_changes.rules_version`, backfilling existing entries as `elo-v1`.
New audit writes must supply the version explicitly.

To tune future games, update `Current()` and bump its version. For a formula
change, update `Rules.Calculate` as well. Preserve old rules in source history
and add regression cases for the new version. Deploy the migration before the
application, with rating writers stopped during the rollout because the old
writer does not supply the now-required rating and version fields. Existing
ratings and audit history are retained; changing rules never replays results.
Any rebase, reset, or historical recalculation must be a separate explicit data
migration. Games use the rules active when their completion is saved.

To change rank boundaries, edit the ascending `Tiers()` table. Ranks are derived
when serving the leaderboard, so threshold changes need no data migration or
frontend threshold edits. New tier IDs should get English and Latvian labels in
the UI's tier-label map; older clients display unfamiliar IDs without rejecting
the leaderboard response. Tier changes do not change historical rating deltas.

Leaderboard cursors include a fingerprint of the scoped participant IDs and
selected scores. The fingerprint and page are read in the same SQL snapshot.
If either scores or membership change between requests, the API returns the
new first page with `reset: true`. The UI discards preceding pages and their
pinned placement, scrolls to the top, and continues with the new cursor. This
also handles legacy cursors without a fingerprint. Player IDs are deduplicated
before rendering as an additional safeguard.
