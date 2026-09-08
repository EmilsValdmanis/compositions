//go:build integration

package database

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/rating"
	"github.com/google/uuid"
)

func TestEloPersistence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	url := startPostgresContainer(t, context.Background())
	if err := RunMigrations(ctx, url, MigrationUp); err != nil {
		t.Fatal(err)
	}
	store, err := NewUserStore(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	users := make([]string, 4)
	for i := range users {
		users[i] = fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1)
		if _, err := store.UpsertUser(ctx, UserRecord{ID: users[i], Name: fmt.Sprintf("Player %d", i), Email: fmt.Sprintf("elo%d@example.com", i)}); err != nil {
			t.Fatal(err)
		}
	}
	game := func() CompletedGameRecord {
		now := time.Now()
		return CompletedGameRecord{ID: uuid.NewString(), RoomCode: "ELO", GameMode: "full", Ranked: true, CompletionKind: "normal", RoundsPlayed: 1, PlayerCount: 2, StartedAt: now, CompletedAt: now.Add(time.Minute), Players: []CompletedGamePlayerRecord{{UserID: users[0], Placement: 1, Won: true}, {UserID: users[1], Placement: 2}}}
	}
	assertRating := func(id string, want, games int) {
		t.Helper()
		var got, count int
		if err := store.pool.QueryRow(ctx, `SELECT rating,games_played FROM player_ratings WHERE user_id=$1`, id).Scan(&got, &count); err != nil {
			t.Fatal(err)
		}
		if got != want || count != games {
			t.Fatalf("%s: rating/games %d/%d; want %d/%d", id, got, count, want, games)
		}
	}
	first := game()
	if err := store.SaveCompletedGame(ctx, first); err != nil {
		t.Fatal(err)
	}
	assertRating(users[0], 1016, 1)
	assertRating(users[1], 984, 1)
	// Retry concurrently: only the first transaction can finalize the game.
	var wg sync.WaitGroup
	errs := make(chan error, 12)
	for range 12 {
		wg.Go(func() { errs <- store.SaveCompletedGame(ctx, first) })
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	assertRating(users[0], 1016, 1)
	var audits int
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM game_rating_changes WHERE game_id=$1`, first.ID).Scan(&audits); err != nil || audits != 2 {
		t.Fatalf("audit count %d: %v", audits, err)
	}
	var before, after int
	if err := store.pool.QueryRow(ctx, `SELECT rating_before,rating_after FROM game_rating_changes WHERE game_id=$1 AND user_id=$2`, first.ID, users[0]).Scan(&before, &after); err != nil || before != 1000 || after != 1016 {
		t.Fatalf("audit %d -> %d: %v", before, after, err)
	}
	var version string
	if err := store.pool.QueryRow(ctx, `SELECT rules_version FROM game_rating_changes WHERE game_id=$1 AND user_id=$2`, first.ID, users[0]).Scan(&version); err != nil || version != rating.Current().Version {
		t.Fatalf("audit rules version %q: %v", version, err)
	}

	for _, scope := range []struct {
		mode   string
		ranked bool
	}{{"quick", true}, {"quick", false}, {"full", false}} {
		g := game()
		g.GameMode = scope.mode
		g.Ranked = scope.ranked
		if err := store.SaveCompletedGame(ctx, g); err != nil {
			t.Fatal(err)
		}
	}
	for _, status := range []string{"technical_abort", "mutual_end", "abandoned"} {
		g := game()
		checkpoint := GameCheckpointRecord{ID: g.ID, RoomCode: g.RoomCode, GameMode: g.GameMode, Ranked: g.Ranked, RoundsPlayed: 1, PlayerCount: 2, StartedAt: g.StartedAt, Players: g.Players}
		if err := store.SaveGameCheckpoint(ctx, checkpoint); err != nil {
			t.Fatal(err)
		}
		if err := store.SaveUnrankedGame(ctx, checkpoint, status, g.CompletedAt); err != nil {
			t.Fatal(err)
		}
		if err := store.SaveCompletedGame(ctx, g); err != nil {
			t.Fatal(err)
		}
	}
	incomplete := game()
	incomplete.Players = incomplete.Players[:1]
	if err := store.SaveCompletedGame(ctx, incomplete); err != nil {
		t.Fatal(err)
	}
	assertRating(users[0], 1016, 1)
	assertRating(users[1], 984, 1)

	// Failure after the first rating lock must roll back the whole completion.
	broken := game()
	broken.Players[1].UserID = uuid.NewString()
	if err := store.SaveCompletedGame(ctx, broken); err == nil {
		t.Fatal("accepted missing user")
	}
	assertRating(users[0], 1016, 1)
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM games WHERE id=$1`, broken.ID).Scan(&audits); err != nil || audits != 0 {
		t.Fatalf("partial game persisted: %d, %v", audits, err)
	}

	// Fail lifetime persistence after all Elo writes to verify atomic rollback.
	if _, err := store.pool.Exec(ctx, `CREATE FUNCTION reject_statistics_test() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test failure'; END $$;
 CREATE TRIGGER reject_statistics_test BEFORE INSERT OR UPDATE ON player_statistics FOR EACH ROW EXECUTE FUNCTION reject_statistics_test()`); err != nil {
		t.Fatal(err)
	}
	failed := game()
	if err := store.SaveCompletedGame(ctx, failed); err == nil {
		t.Fatal("expected injected lifetime failure")
	}
	assertRating(users[0], 1016, 1)
	assertRating(users[1], 984, 1)
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM game_rating_changes WHERE game_id=$1`, failed.ID).Scan(&audits); err != nil || audits != 0 {
		t.Fatalf("partial ratings persisted: %d, %v", audits, err)
	}
	if _, err := store.pool.Exec(ctx, `DROP TRIGGER reject_statistics_test ON player_statistics; DROP FUNCTION reject_statistics_test()`); err != nil {
		t.Fatal(err)
	}

	forfeited := game()
	forfeited.CompletionKind = "forfeit"
	forfeited.Players[1].Forfeited = true
	if err := store.SaveCompletedGame(ctx, forfeited); err != nil {
		t.Fatal(err)
	}
	assertRating(users[0], 1031, 2)
	assertRating(users[1], 969, 2)

	// Different games share players and arrive in opposite seat orders.
	errs = make(chan error, 10)
	for i := range 10 {
		g := game()
		if i%2 == 1 {
			g.Players[0], g.Players[1] = g.Players[1], g.Players[0]
		}
		wg.Go(func() { errs <- store.SaveCompletedGame(ctx, g) })
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var count, sum int
	if err := store.pool.QueryRow(ctx, `SELECT sum(games_played),sum(rating) FROM player_ratings`).Scan(&count, &sum); err != nil || count != 24 || sum != 2000 {
		t.Fatalf("concurrent totals: %d/%d %v", count, sum, err)
	}
	// Audit records must form an unbroken chain of serial updates.
	rows, err := store.pool.Query(ctx, `SELECT rating_before,rating_after FROM game_rating_changes WHERE user_id=$1 ORDER BY rating_before`, users[0])
	if err != nil {
		t.Fatal(err)
	}
	previous := 1000
	for rows.Next() {
		if err := rows.Scan(&before, &after); err != nil {
			t.Fatal(err)
		}
		if before != previous {
			t.Fatalf("broken audit chain %d != %d", before, previous)
		}
		previous = after
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()

	// A four-player game honors shared runner-up placements.
	g := game()
	g.PlayerCount = 4
	g.Players = []CompletedGamePlayerRecord{{UserID: users[0], Placement: 1, Won: true}, {UserID: users[1], Placement: 2}, {UserID: users[2], Placement: 2}, {UserID: users[3], Placement: 4}}
	if err := store.SaveCompletedGame(ctx, g); err != nil {
		t.Fatal(err)
	}
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM game_rating_changes WHERE game_id=$1`, g.ID).Scan(&audits); err != nil || audits != 4 {
		t.Fatalf("four-player audits %d: %v", audits, err)
	}

	// Rating ordering, UUID ties, page boundaries, and off-page placement.
	for i, id := range users {
		value := 1200
		if i == 3 {
			value = 900
		}
		if _, err := store.pool.Exec(ctx, `UPDATE player_ratings SET rating=$2 WHERE user_id=$1`, id, value); err != nil {
			t.Fatal(err)
		}
	}
	page, err := store.GetLeaderboard(ctx, nil, 2, users[3], LeaderboardMetricElo, LeaderboardScopeGlobal)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Players) != 2 || page.Players[0].PlayerID != users[0] || page.Players[1].PlayerID != users[1] || page.NextCursor == nil || page.Placement == nil || page.Placement.Rank != 4 || page.Placement.Score != 900 {
		t.Fatalf("first page %+v", page)
	}
	page, err = store.GetLeaderboard(ctx, page.NextCursor, 2, users[3], LeaderboardMetricElo, LeaderboardScopeGlobal)
	if err != nil || len(page.Players) != 2 || page.Players[0].PlayerID != users[2] || page.Players[1].PlayerID != users[3] || page.NextCursor != nil {
		t.Fatalf("second page %+v: %v", page, err)
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO friendships (user_a_id,user_b_id) VALUES ($1,$2)`, users[0], users[3]); err != nil {
		t.Fatal(err)
	}
	page, err = store.GetLeaderboard(ctx, nil, 50, users[3], LeaderboardMetricElo, LeaderboardScopeFriends)
	if err != nil || len(page.Players) != 2 || page.Players[0].PlayerID != users[0] || page.Placement.Rank != 2 {
		t.Fatalf("friends %+v: %v", page, err)
	}
	// Existing players without a rating row start at the same 1000 baseline.
	if _, err := store.pool.Exec(ctx, `DELETE FROM player_ratings WHERE user_id=$1`, users[3]); err != nil {
		t.Fatal(err)
	}
	page, err = store.GetLeaderboard(ctx, nil, 50, users[3], LeaderboardMetricElo, LeaderboardScopeGlobal)
	if err != nil || page.Placement.Score != 1000 {
		t.Fatalf("baseline %+v: %v", page, err)
	}
	// Downgrading/reapplying the version migration preserves every rating and
	// backfills legacy audit rows with v1, without replaying completed games.
	var auditCount int
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM game_rating_changes`).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if err := RunMigrations(ctx, url, MigrationDown); err != nil {
		t.Fatal(err)
	}
	if err := RunMigrations(ctx, url, MigrationUp); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveCompletedGame(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM game_rating_changes WHERE rules_version = 'elo-v1'`).Scan(&count); err != nil || count != auditCount {
		t.Fatalf("audit history changed on version migration: %d != %d: %v", count, auditCount, err)
	}
	// Removing/recreating the Elo tables preserves game history and statistics.
	var historyCount int
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM games`).Scan(&historyCount); err != nil {
		t.Fatal(err)
	}
	if err := RunMigrations(ctx, url, MigrationDown); err != nil {
		t.Fatal(err)
	}
	if err := RunMigrations(ctx, url, MigrationDown); err != nil {
		t.Fatal(err)
	}
	if err := RunMigrations(ctx, url, MigrationUp); err != nil {
		t.Fatal(err)
	}
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM games`).Scan(&count); err != nil || count != historyCount {
		t.Fatalf("history changed on migration: %d != %d: %v", count, historyCount, err)
	}
	if err := store.SaveCompletedGame(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM game_rating_changes`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("historical game rated: %d: %v", count, err)
	}

}

func TestEloPaginationRestartsAfterRatingChange(t *testing.T) {
	ctx := context.Background()
	dbURL := startPostgresContainer(t, ctx)
	if err := RunMigrations(ctx, dbURL, MigrationUp); err != nil {
		t.Fatal(err)
	}
	store, err := NewUserStore(ctx, dbURL)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ids := []string{uuid.NewString(), uuid.NewString(), uuid.NewString()}
	for i, id := range ids {
		if _, err := store.UpsertUser(ctx, UserRecord{ID: id, Name: fmt.Sprintf("Player%d", i), Email: fmt.Sprintf("audit%d@example.com", i)}); err != nil {
			t.Fatal(err)
		}
	}
	// B is an existing participant who has not played since Elo launched.
	if _, err := store.pool.Exec(ctx, `INSERT INTO player_statistics (user_id,game_mode,ranked,games_played) VALUES ($1,'full',true,1)`, ids[1]); err != nil {
		t.Fatal(err)
	}
	save := func(winner, loser string) {
		now := time.Now()
		err := store.SaveCompletedGame(ctx, CompletedGameRecord{ID: uuid.NewString(), RoomCode: "AUDIT", GameMode: "full", Ranked: true, CompletionKind: "normal", RoundsPlayed: 1, PlayerCount: 2, StartedAt: now, CompletedAt: now.Add(time.Minute), Players: []CompletedGamePlayerRecord{{UserID: winner, Placement: 1, Won: true}, {UserID: loser, Placement: 2}}})
		if err != nil {
			t.Fatal(err)
		}
	}
	save(ids[0], ids[2]) // A=1016, B=1000, C=984
	first, err := store.GetLeaderboard(ctx, nil, 2, ids[1], LeaderboardMetricElo, LeaderboardScopeGlobal)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Players) != 2 || first.NextCursor == nil {
		t.Fatal("invalid fixture")
	}
	save(ids[2], ids[0]) // C=1001, B=1000, A=999
	second, err := store.GetLeaderboard(ctx, first.NextCursor, 2, ids[1], LeaderboardMetricElo, LeaderboardScopeGlobal)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Reset {
		t.Fatal("stale Elo cursor did not restart pagination")
	}
	if len(second.Players) != 2 || second.Players[0].PlayerID != ids[2] || second.Players[1].PlayerID != ids[1] {
		t.Fatalf("fresh page = %+v", second)
	}
	if second.NextCursor == nil || second.NextCursor.Revision == first.NextCursor.Revision {
		t.Fatal("new cursor must identify the new ranking")
	}
	third, err := store.GetLeaderboard(ctx, second.NextCursor, 2, ids[1], LeaderboardMetricElo, LeaderboardScopeGlobal)
	if err != nil {
		t.Fatal(err)
	}
	if third.Reset || third.NextCursor != nil || len(third.Players) != 1 || third.Players[0].PlayerID != ids[0] {
		t.Fatalf("final page = %+v", third)
	}
	seen := map[string]bool{}
	for _, p := range append(second.Players, third.Players...) {
		if seen[p.PlayerID] {
			t.Fatalf("duplicate player %s", p.Name)
		}
		seen[p.PlayerID] = true
	}
	if len(seen) != 3 {
		t.Fatal("missing players")
	}
	// Legacy cursors without a revision safely restart too.
	legacy := *second.NextCursor
	legacy.Revision = ""
	page, err := store.GetLeaderboard(ctx, &legacy, 2, ids[1], LeaderboardMetricElo, LeaderboardScopeGlobal)
	if err != nil || !page.Reset || len(page.Players) != 2 || page.Players[0].PlayerID != ids[2] {
		t.Fatalf("legacy cursor: %+v %v", page, err)
	}
	// Membership changes invalidate the ranking even without rating changes.
	if _, err := store.pool.Exec(ctx, `DELETE FROM player_statistics WHERE user_id=$1`, ids[0]); err != nil {
		t.Fatal(err)
	}
	page, err = store.GetLeaderboard(ctx, second.NextCursor, 2, ids[1], LeaderboardMetricElo, LeaderboardScopeGlobal)
	if err != nil || !page.Reset || len(page.Players) != 2 || page.NextCursor != nil {
		t.Fatalf("membership change: %+v %v", page, err)
	}
}
