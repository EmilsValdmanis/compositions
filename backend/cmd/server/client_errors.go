package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/EmilsValdmanis/compositions/internal/game"
)

const clientErrorInternal = "internal_error"

type httpErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeHTTPError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(httpErrorResponse{Code: code, Message: message})
}

var gameClientErrorCodes = []struct {
	err  error
	code string
}{
	{game.ErrGameInProgress, "game_in_progress"},
	{game.ErrGameNotInProgress, "game_not_in_progress"},
	{game.ErrCannotStartNextRound, "cannot_start_next_round"},
	{game.ErrGameFull, "room_full"},
	{game.ErrNotEnoughPlayers, "not_enough_players"},
	{game.ErrInvalidComposition, "invalid_composition"},
	{game.ErrPlayerAlreadyDrew, "already_drew"},
	{game.ErrPlayerHasntDrawn, "not_drawn"},
	{game.ErrCannotTakeDiscardCard, "cannot_take_discard"},
	{game.ErrCardsNotInHand, "cards_not_in_hand"},
	{game.ErrInitialPointsNotMet, "initial_points"},
	{game.ErrInitialPlayRequiresOwnComp, "initial_own_composition"},
	{game.ErrMustKeepDiscardCard, "keep_final_discard"},
	{game.ErrMustUseDrawnDiscardCard, "use_drawn_discard"},
	{game.ErrInvalidDealingType, "invalid_dealing_type"},
	{game.ErrInvalidDealingOrder, "invalid_dealing_order"},
	{game.ErrInvalidCutSize, "invalid_cut_size"},
	{game.ErrInvalidGameMode, "invalid_game_mode"},
	{game.ErrPlayerNotFound, "player_not_found"},
	{game.ErrPlayerAlreadyForfeited, "already_forfeited"},
}

var exactClientErrorCodes = map[string]string{
	"game already in progress":                           "game_in_progress",
	"game is not in progress":                            "game_not_in_progress",
	"cannot start next round":                            "cannot_start_next_round",
	"game is full":                                       "room_full",
	"need at least 2 players to start":                   "not_enough_players",
	"not a valid composition":                            "invalid_composition",
	"player already drew":                                "already_drew",
	"player hasnt drawn a card yet":                      "not_drawn",
	"cannot take discard card":                           "cannot_take_discard",
	"one or more cards not in hand":                      "cards_not_in_hand",
	"initial compositions must total at least 40 points": "initial_points",
	"initial play requires at least one new composition": "initial_own_composition",
	"player must keep one card for the final discard":    "keep_final_discard",
	"player must use the drawn discard card before playing other cards or discarding": "use_drawn_discard",
	"invalid dealing type":                          "invalid_dealing_type",
	"invalid dealing order":                         "invalid_dealing_order",
	"invalid cut size":                              "invalid_cut_size",
	"invalid game mode":                             "invalid_game_mode",
	"player not found":                              "player_not_found",
	"player already forfeited":                      "already_forfeited",
	"rate limit exceeded":                           "rate_limit_exceeded",
	"authentication required":                       "auth_required",
	"session not found":                             "session_not_found",
	"session belongs to a different user":           "session_user_mismatch",
	"session not active on this connection":         "session_not_found",
	"connect first":                                 "connect_first",
	"already connected":                             "already_connected",
	"missing data":                                  "missing_data",
	"invalid data":                                  "invalid_data",
	"unknown message type":                          "unknown_message_type",
	"already in a room":                             "already_in_room",
	"room not found":                                "room_not_found",
	"game is already starting":                      "game_starting",
	"game already started":                          "game_already_started",
	"room is full":                                  "room_full",
	"join a room first":                             "join_room_first",
	"only the host can start the game":              "host_only_start",
	"all players must be connected":                 "players_disconnected",
	"dealing choice already pending":                "deal_choice_pending",
	"no dealing choice is pending":                  "no_deal_choice",
	"only the deal chooser can choose dealing type": "deal_chooser_only",
	"cut size is required":                          "cut_size_required",
	"invalid dealer":                                "invalid_dealing_order",
	"can only leave in lobby":                       "lobby_only_leave",
	"only the host can start the next round":        "host_only_next_round",
	"player is not active":                          "player_inactive",
	"an end-game request is already active":         "end_request_active",
	"wait before starting another end-game request": "end_request_cooldown",
	"unknown end-game request type":                 "unknown_end_request",
	"end-game request not found or expired":         "end_request_not_found",
	"player is not eligible to vote":                "vote_ineligible",
	"unknown emote":                                 "unknown_emote",
	"unknown draw source":                           "unknown_draw_source",
	"not your turn":                                 "not_your_turn",
	"game state not initialized":                    "game_not_initialized",
	"game is not waiting for a dealing choice":      "no_deal_choice",
	"invalid card rank":                             "invalid_card",
	"invalid card suit":                             "invalid_card",
	"discard card is required":                      "invalid_card",
	"draft card placements must match cards":        "invalid_draft",
	"invalid draft card insert index":               "invalid_draft",
	"invalid draft reclaim joker index":             "invalid_draft",
	"name is required":                              "name_required",
	"problem description is required":               "problem_required",
	"cannot send a request to yourself":             "friend_request_self",
	"user not found":                                "social_user_not_found",
	"friend relationship already exists":            "friend_relationship_exists",
	"friend request not found":                      "friend_request_not_found",
	"game invite not found or expired":              "game_invite_not_found",
	"players are not friends":                       "not_friends",
	"friend is not available":                       "friend_unavailable",
	"friend is not in an active game":               "friend_not_in_game",
	"cannot spectate yourself":                      "cannot_spectate_self",
	"leave your room before spectating":             "spectate_while_in_room",
	"social features are unavailable":               "social_unavailable",
}

func clientErrorCode(err error) string {
	if err == nil {
		return clientErrorInternal
	}

	for _, item := range gameClientErrorCodes {
		if errors.Is(err, item.err) {
			return item.code
		}
	}

	message := err.Error()
	if code, ok := exactClientErrorCodes[message]; ok {
		return code
	}
	if strings.HasPrefix(message, "problem description must be ") {
		return "problem_too_long"
	}

	return clientErrorInternal
}

func clientErrorMessage(err error) string {
	if clientErrorCode(err) == clientErrorInternal {
		return "internal server error"
	}
	return err.Error()
}
