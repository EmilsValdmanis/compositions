package main

import (
	"testing"

	"github.com/EmilsValdmanis/compositions/internal/database"
	"github.com/gorilla/websocket"
)

func TestSocialEventReflectsLivePresence(t *testing.T) {
	presence := newSocialPresence()
	server := &wsServer{socialPresence: presence}
	friendConnection := &websocket.Conn{}
	presence.add("friend-1", friendConnection)

	event := server.socialEventFromRecord("viewer-1", database.SocialSnapshotRecord{
		Friends: []database.SocialUserRecord{{ID: "friend-1", Name: "Avery"}},
		IncomingFriendRequests: []database.FriendRequestRecord{{
			ID:   "request-1",
			User: database.SocialUserRecord{ID: "requester-1", Name: "Blake"},
		}},
	})

	if len(event.Friends) != 1 || !event.Friends[0].Online {
		t.Fatalf("friends = %#v; want online friend", event.Friends)
	}
	if len(event.IncomingFriendRequests) != 1 || event.IncomingFriendRequests[0].User.Online {
		t.Fatalf("requests = %#v; want offline requester", event.IncomingFriendRequests)
	}

	if userID, wentOffline := presence.remove(friendConnection); userID != "friend-1" || !wentOffline {
		t.Fatalf("remove() = (%q, %t); want (friend-1, true)", userID, wentOffline)
	}
	if presence.isOnline("friend-1") {
		t.Fatal("friend remains online after their last connection is removed")
	}
}

func TestSocialPresenceKeepsUserOnlineAcrossConnections(t *testing.T) {
	presence := newSocialPresence()
	first := &websocket.Conn{}
	second := &websocket.Conn{}
	presence.add("user-1", first)
	presence.add("user-1", second)

	if _, wentOffline := presence.remove(first); wentOffline {
		t.Fatal("first disconnect marked a user with another connection offline")
	}
	if !presence.isOnline("user-1") {
		t.Fatal("user should remain online while a second connection is active")
	}
	if _, wentOffline := presence.remove(second); !wentOffline {
		t.Fatal("last disconnect did not mark the user offline")
	}
}
