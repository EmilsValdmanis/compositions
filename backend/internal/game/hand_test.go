package game

import "testing"

func TestHandDraw(t *testing.T) {
	deck := NewGameDeck()
	deck.Shuffle()
	initialDeckSize := len(deck.cards)

	hand := NewHand()

	drew := hand.Draw(deck)

	if !drew {
		t.Fatal("Draw() returned false; expected true")
	}

	if len(hand.cards) != 1 {
		t.Errorf("hand.cards length = %d; want 1", len(hand.cards))
	}

	if len(deck.cards) != initialDeckSize-1 {
		t.Errorf("deck.cards length = %d; want %d", len(deck.cards), initialDeckSize-1)
	}
}

func TestHandDrawFromEmptyDeck(t *testing.T) {
	deck := &Deck{cards: []Card{}}
	hand := NewHand()

	drew := hand.Draw(deck)

	if drew {
		t.Error("Draw() returned true for empty deck; expected false")
	}

	if len(hand.cards) != 0 {
		t.Errorf("hand.cards length = %d; want 0", len(hand.cards))
	}
}

func TestHandDrawMultiple(t *testing.T) {
	deck := NewGameDeck()
	hand := NewHand()

	for i := 1; i <= 5; i++ {
		hand.Draw(deck)

		if len(hand.cards) != i {
			t.Errorf("after %d draws, hand.cards length = %d; want %d", i, len(hand.cards), i)
		}
	}

	if len(deck.cards) != 108-5 {
		t.Errorf("deck.cards length = %d; want %d", len(deck.cards), 108-5)
	}
}
