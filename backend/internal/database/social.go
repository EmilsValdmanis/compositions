package database

import (
	"context"
	"encoding/json"
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
	records, err := s.ListSocialSnapshots(ctx, []string{userID})
	if err != nil {
		return SocialSnapshotRecord{}, err
	}
	return records[strings.TrimSpace(userID)], nil
}

func (s *UserStore) ListSocialSnapshots(ctx context.Context, userIDs []string) (map[string]SocialSnapshotRecord, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("user store is not configured")
	}
	viewerIDs := make([]pgtype.UUID, 0, len(userIDs))
	for _, userID := range userIDs {
		viewerID, err := parseUUID(strings.TrimSpace(userID))
		if err != nil {
			return nil, err
		}
		viewerIDs = append(viewerIDs, viewerID)
	}
	results := make(map[string]SocialSnapshotRecord, len(userIDs))
	if len(viewerIDs) == 0 {
		return results, nil
	}

	rows, err := s.pool.Query(ctx, `
		SELECT viewer_id::text,
			COALESCE((
				SELECT jsonb_agg(jsonb_build_object('ID', u.id::text, 'Name', u.name, 'ImageURL', u.image_url) ORDER BY LOWER(u.name), u.id)
				FROM friendships f
				JOIN users u ON u.id = CASE WHEN f.user_a_id = viewer_id THEN f.user_b_id ELSE f.user_a_id END
				WHERE f.user_a_id = viewer_id OR f.user_b_id = viewer_id
			), '[]'::jsonb),
			COALESCE((
				SELECT jsonb_agg(jsonb_build_object(
					'ID', fr.id::text,
					'User', jsonb_build_object('ID', u.id::text, 'Name', u.name, 'ImageURL', u.image_url),
					'CreatedAt', fr.created_at
				) ORDER BY fr.created_at DESC)
				FROM friend_requests fr JOIN users u ON u.id = fr.sender_id
				WHERE fr.recipient_id = viewer_id
			), '[]'::jsonb),
			COALESCE((
				SELECT jsonb_agg(fr.recipient_id::text ORDER BY fr.created_at DESC)
				FROM friend_requests fr WHERE fr.sender_id = viewer_id
			), '[]'::jsonb),
			COALESCE((
				SELECT jsonb_agg(jsonb_build_object(
					'ID', gi.id::text,
					'User', jsonb_build_object('ID', u.id::text, 'Name', u.name, 'ImageURL', u.image_url),
					'RoomCode', gi.room_code, 'CreatedAt', gi.created_at, 'ExpiresAt', gi.expires_at
				) ORDER BY gi.created_at DESC)
				FROM game_invites gi JOIN users u ON u.id = gi.sender_id
				WHERE gi.recipient_id = viewer_id AND gi.expires_at > NOW()
			), '[]'::jsonb)
		FROM unnest($1::uuid[]) AS viewer_id
	`, viewerIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userID string
		var friendsJSON, requestsJSON, outgoingJSON, invitesJSON json.RawMessage
		if err := rows.Scan(&userID, &friendsJSON, &requestsJSON, &outgoingJSON, &invitesJSON); err != nil {
			return nil, err
		}
		snapshot := SocialSnapshotRecord{}
		if err := json.Unmarshal(friendsJSON, &snapshot.Friends); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(requestsJSON, &snapshot.IncomingFriendRequests); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(outgoingJSON, &snapshot.OutgoingFriendRequestUserIDs); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(invitesJSON, &snapshot.GameInvites); err != nil {
			return nil, err
		}
		results[userID] = snapshot
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
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
