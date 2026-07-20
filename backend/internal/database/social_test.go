//go:build integration

package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUserStoreSocialWorkflow(t *testing.T) {
	ctx := context.Background()
	databaseURL := startPostgresContainer(t, ctx)
	if err := RunMigrations(ctx, databaseURL, MigrationUp); err != nil {
		t.Fatalf("RunMigrations(up) error = %v", err)
	}
	store, err := NewUserStore(ctx, databaseURL)
	if err != nil {
		t.Fatalf("NewUserStore() error = %v", err)
	}
	defer store.Close()

	avery, err := store.UpsertUser(ctx, UserRecord{
		ID: uuid.NewString(), Name: "Avery", Email: "avery@example.com",
	})
	if err != nil {
		t.Fatalf("UpsertUser(Avery) error = %v", err)
	}
	blake, err := store.UpsertUser(ctx, UserRecord{
		ID: uuid.NewString(), Name: "Blake", Email: "blake@example.com",
	})
	if err != nil {
		t.Fatalf("UpsertUser(Blake) error = %v", err)
	}
	casey, err := store.UpsertUser(ctx, UserRecord{
		ID: uuid.NewString(), Name: "Casey", Email: "casey@example.com",
	})
	if err != nil {
		t.Fatalf("UpsertUser(Casey) error = %v", err)
	}

	request, err := store.SendFriendRequest(ctx, avery.ID, blake.ID)
	if err != nil {
		t.Fatalf("SendFriendRequest() error = %v", err)
	}
	if _, err := store.SendFriendRequest(ctx, blake.ID, avery.ID); !errors.Is(err, ErrSocialRelationshipExists) {
		t.Fatalf("SendFriendRequest(reverse duplicate) error = %v; want %v", err, ErrSocialRelationshipExists)
	}

	blakeSocial, err := store.ListSocialSnapshot(ctx, blake.ID)
	if err != nil {
		t.Fatalf("ListSocialSnapshot(Blake pending) error = %v", err)
	}
	if len(blakeSocial.IncomingFriendRequests) != 1 || blakeSocial.IncomingFriendRequests[0].User.ID != avery.ID {
		t.Fatalf("incoming requests = %#v; want request from Avery", blakeSocial.IncomingFriendRequests)
	}

	senderID, err := store.RespondFriendRequest(ctx, blake.ID, request.ID, true)
	if err != nil {
		t.Fatalf("RespondFriendRequest(accept) error = %v", err)
	}
	if senderID != avery.ID {
		t.Fatalf("accepted sender id = %q; want %q", senderID, avery.ID)
	}
	blakeSocial, err = store.ListSocialSnapshot(ctx, blake.ID)
	if err != nil {
		t.Fatalf("ListSocialSnapshot(Blake friends) error = %v", err)
	}
	if len(blakeSocial.Friends) != 1 || blakeSocial.Friends[0].ID != avery.ID {
		t.Fatalf("friends = %#v; want Avery", blakeSocial.Friends)
	}
	if len(blakeSocial.IncomingFriendRequests) != 0 {
		t.Fatalf("incoming requests after accept = %#v; want none", blakeSocial.IncomingFriendRequests)
	}
	if err := store.RemoveFriend(ctx, blake.ID, avery.ID); err != nil {
		t.Fatalf("RemoveFriend() error = %v", err)
	}
	if err := store.RemoveFriend(ctx, blake.ID, avery.ID); !errors.Is(err, ErrUsersNotFriends) {
		t.Fatalf("RemoveFriend(already removed) error = %v; want %v", err, ErrUsersNotFriends)
	}
	if _, err := store.SendFriendRequest(ctx, avery.ID, blake.ID); err != nil {
		t.Fatalf("SendFriendRequest(after remove) error = %v", err)
	}
	if _, err := store.RespondFriendRequest(ctx, blake.ID, request.ID, true); err == nil {
		t.Fatal("RespondFriendRequest(stale request) error = nil; want not found")
	}
	refriendRequest, err := store.ListSocialSnapshot(ctx, blake.ID)
	if err != nil {
		t.Fatalf("ListSocialSnapshot(Blake refriend request) error = %v", err)
	}
	if len(refriendRequest.IncomingFriendRequests) != 1 {
		t.Fatalf("incoming requests after remove = %#v; want one", refriendRequest.IncomingFriendRequests)
	}
	if _, err := store.RespondFriendRequest(ctx, blake.ID, refriendRequest.IncomingFriendRequests[0].ID, true); err != nil {
		t.Fatalf("RespondFriendRequest(refriend) error = %v", err)
	}

	for _, player := range []struct {
		id   string
		wins int
	}{{avery.ID, 3}, {blake.ID, 2}, {casey.ID, 10}} {
		if _, err := store.pool.Exec(ctx, `
			INSERT INTO player_statistics (user_id, games_played, games_won)
			VALUES ($1, 12, $2)
		`, player.id, player.wins); err != nil {
			t.Fatalf("insert statistics for %s: %v", player.id, err)
		}
	}
	friendsLeaderboard, err := store.GetLeaderboard(ctx, nil, 50, avery.ID, LeaderboardMetricWins, LeaderboardScopeFriends)
	if err != nil {
		t.Fatalf("GetLeaderboard(friends) error = %v", err)
	}
	if len(friendsLeaderboard.Players) != 2 || friendsLeaderboard.Players[0].PlayerID != avery.ID || friendsLeaderboard.Players[1].PlayerID != blake.ID {
		t.Fatalf("friends leaderboard = %#v; want Avery and Blake", friendsLeaderboard.Players)
	}
	if friendsLeaderboard.Placement == nil || friendsLeaderboard.Placement.Rank != 1 {
		t.Fatalf("friends placement = %#v; want rank 1", friendsLeaderboard.Placement)
	}
	globalLeaderboard, err := store.GetLeaderboard(ctx, nil, 50, avery.ID, LeaderboardMetricWins, LeaderboardScopeGlobal)
	if err != nil {
		t.Fatalf("GetLeaderboard(global) error = %v", err)
	}
	if len(globalLeaderboard.Players) != 3 || globalLeaderboard.Players[0].PlayerID != casey.ID {
		t.Fatalf("global leaderboard = %#v; want Casey first and three players", globalLeaderboard.Players)
	}
	if globalLeaderboard.Placement == nil || globalLeaderboard.Placement.Rank != 2 {
		t.Fatalf("global placement = %#v; want rank 2", globalLeaderboard.Placement)
	}

	invite, err := store.SendGameInvite(ctx, avery.ID, blake.ID, "ROOM42", time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("SendGameInvite() error = %v", err)
	}
	blakeSocial, err = store.ListSocialSnapshot(ctx, blake.ID)
	if err != nil {
		t.Fatalf("ListSocialSnapshot(Blake invited) error = %v", err)
	}
	if len(blakeSocial.GameInvites) != 1 || blakeSocial.GameInvites[0].RoomCode != "ROOM42" {
		t.Fatalf("game invites = %#v; want ROOM42", blakeSocial.GameInvites)
	}
	if deletedSenderID, err := store.DeleteGameInvite(ctx, blake.ID, invite.ID); err != nil || deletedSenderID != avery.ID {
		t.Fatalf("DeleteGameInvite() = (%q, %v); want (%q, nil)", deletedSenderID, err, avery.ID)
	}
}
