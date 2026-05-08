package game

import "testing"

func benchmarkRun(cards ...Card) *Composition {
	comp, ok := NewRun(cards)
	if !ok {
		panic("benchmark run fixture must be valid")
	}
	return comp
}

func benchmarkDiscardState(hand []Card, discard Card, hasOpened bool, activeComps ...*Composition) *GameState {
	state := newTurnTestState()
	state.players[0].hasOpened = hasOpened
	state.players[0].hand.cards = append([]Card(nil), hand...)
	state.discardPile = &CardPile{cards: []Card{discard}}
	state.activeCompositions = append([]*Composition(nil), activeComps...)
	return state
}

func BenchmarkGameStateCanTakeDiscardNowNoLegalPlay(b *testing.B) {
	baseRun := benchmarkRun(card(Seven, Hearts), card(Eight, Hearts), card(Nine, Hearts))
	state := benchmarkDiscardState(
		[]Card{
			card(Two, Clubs),
			card(Three, Diamonds),
			card(Four, Spades),
			card(Five, Clubs),
			card(Six, Diamonds),
			card(Eight, Spades),
			card(Nine, Clubs),
			card(Jack, Diamonds),
			card(Queen, Spades),
			card(King, Clubs),
			card(Ace, Diamonds),
			card(Two, Hearts),
		},
		card(Five, Spades),
		true,
		baseRun,
	)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if state.canTakeDiscardNow() {
			b.Fatal("canTakeDiscardNow() = true; want false")
		}
	}
}

func BenchmarkGameStateCanTakeDiscardNowOpenedAddition(b *testing.B) {
	baseRun := benchmarkRun(card(Seven, Hearts), card(Eight, Hearts), card(Nine, Hearts))
	state := benchmarkDiscardState(
		[]Card{
			card(Jack, Hearts),
			card(Two, Clubs),
			card(Three, Diamonds),
			card(Four, Spades),
			card(Five, Clubs),
			card(Six, Diamonds),
			card(Eight, Spades),
			card(Nine, Clubs),
			card(Jack, Diamonds),
			card(Queen, Spades),
			card(King, Clubs),
			card(Ace, Diamonds),
		},
		card(Ten, Hearts),
		true,
		baseRun,
	)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !state.canTakeDiscardNow() {
			b.Fatal("canTakeDiscardNow() = false; want true")
		}
	}
}

func BenchmarkGameStateCanTakeDiscardNowOpeningAtForty(b *testing.B) {
	baseRun := benchmarkRun(card(Seven, Hearts), card(Eight, Hearts), card(Nine, Hearts))
	state := benchmarkDiscardState(
		[]Card{
			card(King, Hearts),
			card(King, Diamonds),
			card(King, Clubs),
			card(Two, Spades),
			card(Three, Clubs),
			card(Four, Diamonds),
			card(Five, Spades),
			card(Six, Clubs),
			card(Seven, Diamonds),
			card(Eight, Spades),
			card(Nine, Clubs),
			card(Jack, Diamonds),
		},
		card(Ten, Hearts),
		false,
		baseRun,
	)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !state.canTakeDiscardNow() {
			b.Fatal("canTakeDiscardNow() = false; want true")
		}
	}
}

func BenchmarkGameStateCanTakeDiscardNowJokerReclaim(b *testing.B) {
	baseRun := benchmarkRun(card(Five, Hearts), benchmarkJoker(), card(Seven, Hearts))
	state := benchmarkDiscardState(
		[]Card{
			card(Two, Clubs),
			card(Three, Diamonds),
			card(Four, Spades),
			card(Five, Clubs),
			card(Six, Diamonds),
			card(Eight, Spades),
			card(Nine, Clubs),
			card(Jack, Diamonds),
			card(Queen, Spades),
			card(King, Clubs),
			card(Ace, Diamonds),
			card(Two, Hearts),
		},
		card(Six, Hearts),
		true,
		baseRun,
	)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !state.canTakeDiscardNow() {
			b.Fatal("canTakeDiscardNow() = false; want true")
		}
	}
}

func benchmarkJoker() Card {
	return Card{isJoker: true}
}
