package game

import "testing"

func TestCardPileDrawOne(t *testing.T) {
	pile := &CardPile{cards: []Card{{rank: Ace, suit: Hearts}, {rank: King, suit: Spades}}}

	card, ok := pile.DrawOne()

	if !ok {
		t.Fatal("DrawOne() returned false; expected true")
	}
	if card.rank != Ace || card.suit != Hearts {
		t.Errorf("DrawOne() = %+v; want Ace of Hearts", card)
	}
	if len(pile.cards) != 1 {
		t.Errorf("len(pile.cards) = %d; want 1", len(pile.cards))
	}
}

func TestCardPileDrawOneFromEmptyPile(t *testing.T) {
	pile := &CardPile{cards: []Card{}}

	_, ok := pile.DrawOne()

	if ok {
		t.Error("DrawOne() returned true for empty pile; expected false")
	}
}

func TestCardPileAddToTop(t *testing.T) {
	bottom := Card{rank: Four, suit: Clubs}
	top := Card{rank: Ten, suit: Diamonds}
	pile := &CardPile{cards: []Card{bottom}}

	pile.AddToTop(top)

	card, ok := pile.DrawOne()
	if !ok {
		t.Fatal("DrawOne() returned false; expected true")
	}
	if card != top {
		t.Errorf("DrawOne() after AddToTop() = %+v; want %+v", card, top)
	}
}
