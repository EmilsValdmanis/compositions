package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/EmilsValdmanis/compositions/internal/database"
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

var clientErrorCodes = []struct {
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
	{game.ErrInvalidDealer, "invalid_dealing_order"},
	{database.ErrSessionNotFound, "session_not_found"},
	{database.ErrSocialSelfRequest, "friend_request_self"},
	{database.ErrSocialUserNotFound, "social_user_not_found"},
	{database.ErrSocialRelationshipExists, "friend_relationship_exists"},
	{database.ErrFriendRequestNotFound, "friend_request_not_found"},
	{database.ErrGameInviteNotFound, "game_invite_not_found"},
	{database.ErrUsersNotFriends, "not_friends"},
}

type clientError struct {
	code    string
	message string
}

func (e clientError) Error() string { return e.message }

func describeClientError(err error) clientError {
	if problem, ok := errors.AsType[clientError](err); ok {
		return problem
	}
	for _, item := range clientErrorCodes {
		if errors.Is(err, item.err) {
			return clientError{item.code, item.err.Error()}
		}
	}
	return clientError{clientErrorInternal, "internal server error"}
}
