package database

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrSocialUserNotFound       = errors.New("user not found")
	ErrSocialRelationshipExists = errors.New("friend relationship already exists")
	ErrFriendRequestNotFound    = errors.New("friend request not found")
	ErrGameInviteNotFound       = errors.New("game invite not found or expired")
	ErrUsersNotFriends          = errors.New("players are not friends")
)

const postgresForeignKeyViolation = "23503"

type SocialUserRecord struct {
	ID       string
	Name     string
	ImageURL string
}

type FriendRequestRecord struct {
	ID        string
	User      SocialUserRecord
	CreatedAt time.Time
}

type GameInviteRecord struct {
	ID        string
	User      SocialUserRecord
	RoomCode  string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type SocialSnapshotRecord struct {
	Friends                      []SocialUserRecord
	IncomingFriendRequests       []FriendRequestRecord
	OutgoingFriendRequestUserIDs []string
	GameInvites                  []GameInviteRecord
}

func (s *UserStore) ListSocialSnapshot(ctx context.Context, userID string) (SocialSnapshotRecord, error) {
	if s == nil || s.pool == nil {
		return SocialSnapshotRecord{}, errors.New("user store is not configured")
	}

	viewerID, err := parseUUID(strings.TrimSpace(userID))
	if err != nil {
		return SocialSnapshotRecord{}, err
	}

	snapshot := SocialSnapshotRecord{
		Friends:                      []SocialUserRecord{},
		IncomingFriendRequests:       []FriendRequestRecord{},
		OutgoingFriendRequestUserIDs: []string{},
		GameInvites:                  []GameInviteRecord{},
	}

	friendRows, err := s.pool.Query(ctx, `
		SELECT u.id::text, u.name, u.image_url
		FROM friendships f
		JOIN users u ON u.id = CASE WHEN f.user_a_id = $1 THEN f.user_b_id ELSE f.user_a_id END
		WHERE f.user_a_id = $1 OR f.user_b_id = $1
		ORDER BY LOWER(u.name), u.id
	`, viewerID)
	if err != nil {
		return SocialSnapshotRecord{}, err
	}
	for friendRows.Next() {
		var friend SocialUserRecord
		if err := friendRows.Scan(&friend.ID, &friend.Name, &friend.ImageURL); err != nil {
			friendRows.Close()
			return SocialSnapshotRecord{}, err
		}
		snapshot.Friends = append(snapshot.Friends, friend)
	}
	if err := friendRows.Err(); err != nil {
		friendRows.Close()
		return SocialSnapshotRecord{}, err
	}
	friendRows.Close()

	requestRows, err := s.pool.Query(ctx, `
		SELECT fr.id::text, u.id::text, u.name, u.image_url, fr.created_at
		FROM friend_requests fr
		JOIN users u ON u.id = fr.sender_id
		WHERE fr.recipient_id = $1
		ORDER BY fr.created_at DESC
	`, viewerID)
	if err != nil {
		return SocialSnapshotRecord{}, err
	}
	for requestRows.Next() {
		var request FriendRequestRecord
		if err := requestRows.Scan(&request.ID, &request.User.ID, &request.User.Name, &request.User.ImageURL, &request.CreatedAt); err != nil {
			requestRows.Close()
			return SocialSnapshotRecord{}, err
		}
		snapshot.IncomingFriendRequests = append(snapshot.IncomingFriendRequests, request)
	}
	if err := requestRows.Err(); err != nil {
		requestRows.Close()
		return SocialSnapshotRecord{}, err
	}
	requestRows.Close()

	outgoingRows, err := s.pool.Query(ctx, `
		SELECT recipient_id::text
		FROM friend_requests
		WHERE sender_id = $1
		ORDER BY created_at DESC
	`, viewerID)
	if err != nil {
		return SocialSnapshotRecord{}, err
	}
	for outgoingRows.Next() {
		var recipientID string
		if err := outgoingRows.Scan(&recipientID); err != nil {
			outgoingRows.Close()
			return SocialSnapshotRecord{}, err
		}
		snapshot.OutgoingFriendRequestUserIDs = append(snapshot.OutgoingFriendRequestUserIDs, recipientID)
	}
	if err := outgoingRows.Err(); err != nil {
		outgoingRows.Close()
		return SocialSnapshotRecord{}, err
	}
	outgoingRows.Close()

	inviteRows, err := s.pool.Query(ctx, `
		SELECT gi.id::text, u.id::text, u.name, u.image_url, gi.room_code, gi.created_at, gi.expires_at
		FROM game_invites gi
		JOIN users u ON u.id = gi.sender_id
		WHERE gi.recipient_id = $1 AND gi.expires_at > NOW()
		ORDER BY gi.created_at DESC
	`, viewerID)
	if err != nil {
		return SocialSnapshotRecord{}, err
	}
	for inviteRows.Next() {
		var invite GameInviteRecord
		if err := inviteRows.Scan(&invite.ID, &invite.User.ID, &invite.User.Name, &invite.User.ImageURL, &invite.RoomCode, &invite.CreatedAt, &invite.ExpiresAt); err != nil {
			inviteRows.Close()
			return SocialSnapshotRecord{}, err
		}
		snapshot.GameInvites = append(snapshot.GameInvites, invite)
	}
	if err := inviteRows.Err(); err != nil {
		inviteRows.Close()
		return SocialSnapshotRecord{}, err
	}
	inviteRows.Close()

	return snapshot, nil
}

func (s *UserStore) SendFriendRequest(ctx context.Context, senderID, recipientID string) (FriendRequestRecord, error) {
	if s == nil || s.pool == nil {
		return FriendRequestRecord{}, errors.New("user store is not configured")
	}
	sender, recipient, err := parseSocialUserPair(senderID, recipientID)
	if err != nil {
		return FriendRequestRecord{}, err
	}

	var recipientExists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, recipient).Scan(&recipientExists); err != nil {
		return FriendRequestRecord{}, err
	}
	if !recipientExists {
		return FriendRequestRecord{}, ErrSocialUserNotFound
	}

	var alreadyFriends bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM friendships
			WHERE user_a_id = LEAST($1::uuid, $2::uuid) AND user_b_id = GREATEST($1::uuid, $2::uuid)
		)
	`, sender, recipient).Scan(&alreadyFriends); err != nil {
		return FriendRequestRecord{}, err
	}
	if alreadyFriends {
		return FriendRequestRecord{}, ErrSocialRelationshipExists
	}

	var request FriendRequestRecord
	err = s.pool.QueryRow(ctx, `
		INSERT INTO friend_requests (sender_id, recipient_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
		RETURNING id::text, created_at
	`, sender, recipient).Scan(&request.ID, &request.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return FriendRequestRecord{}, ErrSocialRelationshipExists
	}
	if err != nil {
		return FriendRequestRecord{}, socialConstraintError(err)
	}
	return request, nil
}

func (s *UserStore) RespondFriendRequest(ctx context.Context, recipientID, requestID string, accept bool) (string, error) {
	if s == nil || s.pool == nil {
		return "", errors.New("user store is not configured")
	}
	recipient, err := parseUUID(strings.TrimSpace(recipientID))
	if err != nil {
		return "", err
	}
	request, err := parseUUID(strings.TrimSpace(requestID))
	if err != nil {
		return "", ErrFriendRequestNotFound
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var sender pgtype.UUID
	err = tx.QueryRow(ctx, `
		SELECT sender_id FROM friend_requests
		WHERE id = $1 AND recipient_id = $2
		FOR UPDATE
	`, request, recipient).Scan(&sender)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrFriendRequestNotFound
	}
	if err != nil {
		return "", err
	}

	if accept {
		if _, err := tx.Exec(ctx, `
			INSERT INTO friendships (user_a_id, user_b_id)
			VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
			ON CONFLICT DO NOTHING
		`, sender, recipient); err != nil {
			return "", err
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM friend_requests WHERE id = $1`, request); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return nullableUUIDString(sender), nil
}

func (s *UserStore) RemoveFriend(ctx context.Context, userID, friendID string) error {
	if s == nil || s.pool == nil {
		return errors.New("user store is not configured")
	}
	user, friend, err := parseSocialUserPair(userID, friendID)
	if err != nil {
		return err
	}

	command, err := s.pool.Exec(ctx, `
		DELETE FROM friendships
		WHERE user_a_id = LEAST($1::uuid, $2::uuid)
			AND user_b_id = GREATEST($1::uuid, $2::uuid)
	`, user, friend)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return ErrUsersNotFriends
	}
	return nil
}

func (s *UserStore) SendGameInvite(ctx context.Context, senderID, recipientID, roomCode string, expiresAt time.Time) (GameInviteRecord, error) {
	if s == nil || s.pool == nil {
		return GameInviteRecord{}, errors.New("user store is not configured")
	}
	sender, recipient, err := parseSocialUserPair(senderID, recipientID)
	if err != nil {
		return GameInviteRecord{}, err
	}
	cleanRoomCode := strings.ToUpper(strings.TrimSpace(roomCode))
	if cleanRoomCode == "" {
		return GameInviteRecord{}, errors.New("room code is required")
	}
	if !expiresAt.After(time.Now()) {
		return GameInviteRecord{}, errors.New("game invite expiry must be in the future")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return GameInviteRecord{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var areFriends bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM friendships
			WHERE user_a_id = LEAST($1::uuid, $2::uuid) AND user_b_id = GREATEST($1::uuid, $2::uuid)
		)
	`, sender, recipient).Scan(&areFriends); err != nil {
		return GameInviteRecord{}, err
	}
	if !areFriends {
		return GameInviteRecord{}, ErrUsersNotFriends
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM game_invites
		WHERE expires_at <= NOW() OR (sender_id = $1 AND recipient_id = $2 AND room_code = $3)
	`, sender, recipient, cleanRoomCode); err != nil {
		return GameInviteRecord{}, err
	}

	var invite GameInviteRecord
	err = tx.QueryRow(ctx, `
		INSERT INTO game_invites (sender_id, recipient_id, room_code, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, room_code, created_at, expires_at
	`, sender, recipient, cleanRoomCode, expiresAt).Scan(&invite.ID, &invite.RoomCode, &invite.CreatedAt, &invite.ExpiresAt)
	if err != nil {
		return GameInviteRecord{}, socialConstraintError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return GameInviteRecord{}, err
	}
	return invite, nil
}

func (s *UserStore) GetGameInvite(ctx context.Context, recipientID, inviteID string) (GameInviteRecord, error) {
	if s == nil || s.pool == nil {
		return GameInviteRecord{}, errors.New("user store is not configured")
	}
	recipient, err := parseUUID(strings.TrimSpace(recipientID))
	if err != nil {
		return GameInviteRecord{}, err
	}
	inviteUUID, err := parseUUID(strings.TrimSpace(inviteID))
	if err != nil {
		return GameInviteRecord{}, ErrGameInviteNotFound
	}

	var invite GameInviteRecord
	err = s.pool.QueryRow(ctx, `
		SELECT gi.id::text, gi.sender_id::text, gi.room_code, gi.created_at, gi.expires_at
		FROM game_invites gi
		WHERE gi.id = $1 AND gi.recipient_id = $2 AND gi.expires_at > NOW()
	`, inviteUUID, recipient).Scan(&invite.ID, &invite.User.ID, &invite.RoomCode, &invite.CreatedAt, &invite.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return GameInviteRecord{}, ErrGameInviteNotFound
	}
	if err != nil {
		return GameInviteRecord{}, err
	}
	return invite, nil
}

func (s *UserStore) DeleteGameInvite(ctx context.Context, recipientID, inviteID string) (string, error) {
	if s == nil || s.pool == nil {
		return "", errors.New("user store is not configured")
	}
	recipient, err := parseUUID(strings.TrimSpace(recipientID))
	if err != nil {
		return "", err
	}
	inviteUUID, err := parseUUID(strings.TrimSpace(inviteID))
	if err != nil {
		return "", ErrGameInviteNotFound
	}

	var senderID string
	err = s.pool.QueryRow(ctx, `
		DELETE FROM game_invites
		WHERE id = $1 AND recipient_id = $2
		RETURNING sender_id::text
	`, inviteUUID, recipient).Scan(&senderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrGameInviteNotFound
	}
	return senderID, err
}

func parseSocialUserPair(firstID, secondID string) (pgtype.UUID, pgtype.UUID, error) {
	first, err := parseUUID(strings.TrimSpace(firstID))
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	second, err := parseUUID(strings.TrimSpace(secondID))
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	if first == second {
		return pgtype.UUID{}, pgtype.UUID{}, errors.New("cannot send a request to yourself")
	}
	return first, second, nil
}

func socialConstraintError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == postgresForeignKeyViolation {
		return ErrSocialUserNotFound
	}
	return err
}
