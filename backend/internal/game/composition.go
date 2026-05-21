package game

import "slices"

type compositionVariant string

const (
	set compositionVariant = "set"
	run compositionVariant = "run"
)

type Composition struct {
	variant              compositionVariant
	cards                []Card
	jokerRepresentations map[int][]Card
}

func newComposition(cards []Card, variant compositionVariant, requireOrderedRun bool) (*Composition, bool) {
	c := &Composition{
		variant:              variant,
		cards:                slices.Clone(cards),
		jokerRepresentations: make(map[int][]Card),
	}
	if !c.isValid() {
		return nil, false
	}
	if c.variant == run {
		if requireOrderedRun && !runCardsAreOrdered(c.cards) {
			return nil, false
		}
		c.normalizeRunCards()
	}
	c.assignJokers()
	return c, true
}

func NewComposition(cards []Card, variant compositionVariant) (*Composition, bool) {
	return newComposition(cards, variant, false)
}

func NewSet(cards []Card) (*Composition, bool) {
	return newComposition(cards, set, false)
}

func NewRun(cards []Card) (*Composition, bool) {
	return newComposition(cards, run, true)
}

func (c *Composition) WithAddedCards(cards []Card) (*Composition, bool) {
	combined := make([]Card, 0, len(c.cards)+len(cards))
	combined = append(combined, c.cards...)
	combined = append(combined, cards...)

	return newComposition(combined, c.variant, false)
}

func (c *Composition) ReclaimJoker(cardIndex int, replacement Card) (*Composition, bool) {
	if cardIndex < 0 || cardIndex >= len(c.cards) {
		return nil, false
	}
	if !c.cards[cardIndex].isJoker || replacement.isJoker {
		return nil, false
	}

	representation, ok := c.JokerRepresentation(cardIndex)
	if !ok {
		return nil, false
	}
	if !cardsEqual(representation, replacement) {
		return nil, false
	}

	updatedCards := slices.Clone(c.cards)
	updatedCards[cardIndex] = replacement

	return newComposition(updatedCards, c.variant, false)
}

func (c *Composition) canReclaimJoker(cardIndex int, replacement Card) bool {
	if cardIndex < 0 || cardIndex >= len(c.cards) {
		return false
	}
	if !c.cards[cardIndex].isJoker || replacement.isJoker {
		return false
	}

	options, ok := c.jokerRepresentations[cardIndex]
	if !ok || len(options) != 1 {
		return false
	}

	return cardsEqual(options[0], replacement)
}

func (c *Composition) AddedCardsPoints(cards []Card) (int, bool) {
	extended, ok := c.WithAddedCards(cards)
	if !ok {
		return 0, false
	}

	return extended.Points() - c.Points(), true
}

func (c *Composition) addedCardsPointsNoAlloc(cards []Card, combinedBuf []Card) (int, bool) {
	combined := combinedBuf[:0]
	combined = append(combined, c.cards...)
	combined = append(combined, cards...)

	extended := Composition{variant: c.variant, cards: combined}
	if !extended.isValid() {
		return 0, false
	}

	return extended.Points() - c.Points(), true
}

func (c *Composition) Points() int {
	switch c.variant {
	case set:
		return c.setPoints()
	case run:
		return c.runPoints()
	default:
		return 0
	}
}

func (c *Composition) isComplete() bool {
	switch c.variant {
	case set:
		return c.isCompleteSet()
	case run:
		return c.isCompleteRun()
	default:
		return false
	}
}

func (c *Composition) isCompleteSet() bool {
	if len(c.cards) != 4 {
		return false
	}
	if len(nonJokerCards(c.cards)) != len(c.cards) {
		return false
	}

	setRank := c.cards[0].rank
	seenSuits := make(map[Suit]bool, 4)
	for _, card := range c.cards {
		if card.rank != setRank || seenSuits[card.suit] {
			return false
		}
		seenSuits[card.suit] = true
	}

	return len(seenSuits) == 4
}

func (c *Composition) isCompleteRun() bool {
	if len(c.cards) != 14 {
		return false
	}
	if len(nonJokerCards(c.cards)) != len(c.cards) {
		return false
	}

	runSuit := c.cards[0].suit
	rankCounts := make(map[Rank]int, 13)
	for _, card := range c.cards {
		if card.suit != runSuit {
			return false
		}
		rankCounts[card.rank]++
	}

	if rankCounts[Ace] != 2 {
		return false
	}

	for rank := Two; rank <= King; rank++ {
		if rankCounts[rank] != 1 {
			return false
		}
	}

	return len(rankCounts) == 13
}

func (c *Composition) setPoints() int {
	setRank := c.setRank()

	total := 0
	for i, card := range c.cards {
		if !card.isJoker {
			total += rankPoints(card.rank, false)
			continue
		}

		representation, ok := c.JokerRepresentation(i)
		if ok {
			total += rankPoints(representation.rank, false)
			continue
		}

		total += rankPoints(setRank, false)
	}

	return total
}

func (c *Composition) setRank() Rank {
	for _, card := range c.cards {
		if !card.isJoker {
			return card.rank
		}
	}

	return Ace
}

func (c *Composition) runPoints() int {
	realCards := nonJokerCards(c.cards)
	jokerCount := len(jokerCardIndices(c.cards))
	best := 0

	for _, aceLow := range []bool{false, true} {
		replacements, ok := tryFitSequence(realCards, jokerCount, aceLow)
		if !ok {
			continue
		}

		cards := make([]Card, 0, len(realCards)+len(replacements))
		cards = append(cards, realCards...)
		cards = append(cards, replacements...)

		total := runCardsPoints(cards, aceLow)
		if total > best {
			best = total
		}
	}

	return best
}

func runCardsPoints(cards []Card, aceLow bool) int {
	aceCount := 0
	for _, card := range cards {
		if card.rank == Ace {
			aceCount++
		}
	}

	total := 0
	aceAssignedLow := false
	for _, card := range cards {
		if card.rank != Ace {
			total += rankPoints(card.rank, false)
			continue
		}

		if aceCount == 2 && !aceAssignedLow {
			total += rankPoints(card.rank, true)
			aceAssignedLow = true
			continue
		}

		total += rankPoints(card.rank, aceLow)
	}

	return total
}

func rankPoints(rank Rank, aceLow bool) int {
	if rank == Ace {
		if aceLow {
			return 1
		}
		return 10
	}
	if rank >= Jack && rank <= King {
		return 10
	}
	if rank >= Two && rank <= Ten {
		return int(rank)
	}

	return 0
}

func (c *Composition) isValid() bool {
	switch c.variant {
	case set:
		return c.isValidSet()
	case run:
		return c.isValidRun()
	default:
		return false
	}
}

func (c *Composition) isValidSet() bool {
	cardCount := len(c.cards)
	if cardCount < 3 || cardCount > 4 {
		return false
	}

	var realCards []Card
	for _, card := range c.cards {
		if !card.isJoker {
			realCards = append(realCards, card)
		}
	}

	if len(realCards) > 0 {
		firstRank := realCards[0].rank
		for _, card := range realCards[1:] {
			if card.rank != firstRank {
				return false
			}
		}

		seenSuits := make(map[Suit]bool)
		for _, card := range realCards {
			if seenSuits[card.suit] {
				return false
			}
			seenSuits[card.suit] = true
		}
	}
	return true
}

func (c *Composition) isValidRun() bool {
	if len(c.cards) < 3 || len(c.cards) > 14 {
		return false
	}

	var realCards []Card
	var jokerCount int

	for _, card := range c.cards {
		if card.isJoker {
			jokerCount++
		} else {
			realCards = append(realCards, card)
		}
	}

	if len(realCards) > 0 {
		firstSuit := realCards[0].suit
		for _, card := range realCards[1:] {
			if card.suit != firstSuit {
				return false
			}
		}
	}

	_, ok := tryFitSequence(realCards, jokerCount, false)
	if ok {
		return true
	}

	_, ok = tryFitSequence(realCards, jokerCount, true)
	return ok
}

func (c *Composition) assignJokers() bool {
	if len(c.cards) == 0 {
		c.jokerRepresentations = map[int][]Card{}
		return true
	}

	switch c.variant {
	case set:
		return c.assignSetJokers()
	case run:
		return c.assignRunJokers()
	default:
		return false
	}
}

func (c *Composition) assignSetJokers() bool {
	jokerIndices := jokerCardIndices(c.cards)
	c.jokerRepresentations = make(map[int][]Card, len(jokerIndices))
	if len(jokerIndices) == 0 {
		return true
	}

	realCards := nonJokerCards(c.cards)
	if len(realCards) == 0 {
		allCards := allNaturalCards()
		for _, jokerIndex := range jokerIndices {
			c.jokerRepresentations[jokerIndex] = slices.Clone(allCards)
		}
		return true
	}

	usedSuits := make(map[Suit]bool, len(realCards))
	for _, realCard := range realCards {
		usedSuits[realCard.suit] = true
	}

	options, ok := missingSetCards(realCards[0].rank, usedSuits)
	if !ok {
		return false
	}

	for _, jokerIndex := range jokerIndices {
		c.jokerRepresentations[jokerIndex] = slices.Clone(options)
	}

	return true
}

func (c *Composition) assignRunJokers() bool {
	jokerIndices := jokerCardIndices(c.cards)
	c.jokerRepresentations = make(map[int][]Card, len(jokerIndices))
	if len(jokerIndices) == 0 {
		return true
	}

	_, jokerRepresentations, _, ok := bestRunOrder(c.cards)
	if !ok {
		return false
	}

	for jokerIndex, replacement := range jokerRepresentations {
		c.jokerRepresentations[jokerIndex] = []Card{replacement}
	}

	return true
}

func (c *Composition) normalizeRunCards() {
	ordered, _, _, _ := bestRunOrder(c.cards)
	c.cards = ordered
}

func missingSetCards(rank Rank, usedSuits map[Suit]bool) ([]Card, bool) {
	options := make([]Card, 0, 4-len(usedSuits))
	for _, suit := range []Suit{Hearts, Diamonds, Clubs, Spades} {
		if usedSuits[suit] {
			continue
		}
		options = append(options, Card{rank: rank, suit: suit})
	}

	if len(options) == 0 {
		return nil, false
	}

	return options, true
}

func sequenceRanksForCards(cards []Card, aceLow bool) ([]int, bool) {
	ranks := make([]int, len(cards))
	aceCount := 0
	for _, card := range cards {
		if card.isJoker {
			return nil, false
		}
		if card.rank == Ace {
			aceCount++
		}
	}

	aceAssignedLow := false
	for i, card := range cards {
		if card.rank == Ace {
			if aceCount == 2 && !aceAssignedLow {
				ranks[i] = 1
				aceAssignedLow = true
			} else if aceLow {
				ranks[i] = 1
			} else {
				ranks[i] = 14
			}
			continue
		}

		ranks[i] = int(card.rank)
	}

	return ranks, true
}

func tryFitSequenceRanks(realCards []Card, jokerCount int, aceLow bool) ([]int, bool) {
	if len(realCards) == 0 {
		if jokerCount < 3 || jokerCount > 14 {
			return nil, false
		}

		replacements := make([]int, 0, jokerCount)
		for rank := Ace; len(replacements) < jokerCount && rank <= King; rank++ {
			replacements = append(replacements, int(rank))
		}
		return replacements, true
	}

	ranks, ok := sequenceRanksForCards(realCards, aceLow)
	if !ok {
		return nil, false
	}

	slices.Sort(ranks)

	for i := 1; i < len(ranks); i++ {
		if ranks[i] == ranks[i-1] {
			return nil, false
		}
	}

	replacements := make([]int, 0, jokerCount)
	for i := 1; i < len(ranks); i++ {
		for rank := ranks[i-1] + 1; rank < ranks[i]; rank++ {
			replacements = append(replacements, rank)
		}
	}

	if len(replacements) > jokerCount {
		return nil, false
	}

	remaining := jokerCount - len(replacements)
	for rank := ranks[len(ranks)-1] + 1; remaining > 0 && rank <= 14; rank++ {
		replacements = append(replacements, rank)
		remaining--
	}
	for rank := ranks[0] - 1; remaining > 0 && rank >= 1; rank-- {
		replacements = append(replacements, rank)
		remaining--
	}

	if remaining != 0 {
		return nil, false
	}

	return replacements, true
}

func tryFitSequence(realCards []Card, jokerCount int, aceLow bool) ([]Card, bool) {
	replacementRanks, ok := tryFitSequenceRanks(realCards, jokerCount, aceLow)
	if !ok {
		return nil, false
	}

	runSuit := Hearts
	if len(realCards) > 0 {
		runSuit = realCards[0].suit
	}

	replacements := make([]Card, 0, len(replacementRanks))
	for _, rank := range replacementRanks {
		replacements = append(replacements, Card{rank: sequenceRankToCardRank(rank), suit: runSuit})
	}

	return replacements, true
}

func sameCard(a, b Card) bool {
	return a.rank == b.rank && a.suit == b.suit && a.isJoker == b.isJoker
}

func bestRunOrder(cards []Card) ([]Card, map[int]Card, bool, bool) {
	realCards := nonJokerCards(cards)
	runLength := len(cards)
	runSuit := Hearts
	if len(realCards) > 0 {
		runSuit = realCards[0].suit
	}

	if len(realCards) == 0 {
		ordered := slices.Clone(cards)
		jokerRepresentations := make(map[int]Card, len(cards))
		for index := range ordered {
			jokerRepresentations[index] = Card{rank: Rank(index + 1), suit: runSuit}
		}
		return ordered, jokerRepresentations, true, true
	}

	type runOrderCandidate struct {
		ordered             []Card
		jokerRepresentations map[int]Card
		matchesInput        bool
		matchCount          int
	}

	var best *runOrderCandidate

	for _, aceLow := range []bool{false, true} {
		realRanks, _ := sequenceRanksForCards(realCards, aceLow)

		realCardsByRank := make(map[int]Card, len(realRanks))
		for index, rank := range realRanks {
			realCardsByRank[rank] = realCards[index]
		}

		for start := 1; start <= 15-runLength; start++ {
			end := start + runLength - 1
			missingCount := 0
			valid := true

			for _, rank := range realRanks {
				if rank < start || rank > end {
					valid = false
					break
				}
			}
			if !valid {
				continue
			}

			for rank := start; rank <= end; rank++ {
				if _, ok := realCardsByRank[rank]; !ok {
					missingCount++
				}
			}
			if missingCount != len(cards)-len(realCards) {
				continue
			}

			ordered := make([]Card, 0, len(cards))
			jokerRepresentations := make(map[int]Card, missingCount)
			for rank := start; rank <= end; rank++ {
				if realCard, ok := realCardsByRank[rank]; ok {
					ordered = append(ordered, realCard)
					continue
				}

				jokerIndex := len(ordered)
				ordered = append(ordered, Card{isJoker: true})
				jokerRepresentations[jokerIndex] = Card{rank: sequenceRankToCardRank(rank), suit: runSuit}
			}

			matchCount := 0
			matchesInput := len(ordered) == len(cards)
			for i := range ordered {
				if sameCard(cards[i], ordered[i]) {
					matchCount++
					continue
				}
				matchesInput = false
			}

			candidate := &runOrderCandidate{
				ordered:              ordered,
				jokerRepresentations: jokerRepresentations,
				matchesInput:         matchesInput,
				matchCount:           matchCount,
			}

			if best == nil || candidate.matchesInput || candidate.matchCount > best.matchCount {
				best = candidate
			}
			if candidate.matchesInput {
				return candidate.ordered, candidate.jokerRepresentations, true, true
			}
		}
	}

	if best == nil {
		return nil, nil, false, false
	}

	return best.ordered, best.jokerRepresentations, false, true
}

func runCardsAreOrdered(cards []Card) bool {
	_, _, ordered, ok := bestRunOrder(cards)
	return ok && ordered
}

func (c *Composition) JokerRepresentation(cardIndex int) (Card, bool) {
	options, ok := c.JokerRepresentations(cardIndex)
	if !ok || len(options) != 1 {
		return Card{}, false
	}

	return options[0], true
}

func (c *Composition) JokerRepresentations(cardIndex int) ([]Card, bool) {
	if cardIndex < 0 || cardIndex >= len(c.cards) {
		return nil, false
	}
	if !c.cards[cardIndex].isJoker {
		return nil, false
	}

	options, ok := c.jokerRepresentations[cardIndex]
	if !ok {
		return nil, false
	}

	return slices.Clone(options), true
}

func jokerCardIndices(cards []Card) []int {
	indices := make([]int, 0, len(cards))
	for i, card := range cards {
		if card.isJoker {
			indices = append(indices, i)
		}
	}
	return indices
}

func nonJokerCards(cards []Card) []Card {
	realCards := make([]Card, 0, len(cards))
	for _, card := range cards {
		if !card.isJoker {
			realCards = append(realCards, card)
		}
	}
	return realCards
}

func allNaturalCards() []Card {
	allCards := make([]Card, 0, 52)
	for _, suit := range []Suit{Hearts, Diamonds, Clubs, Spades} {
		for rank := Ace; rank <= King; rank++ {
			allCards = append(allCards, Card{rank: rank, suit: suit})
		}
	}
	return allCards
}

func sequenceRankToCardRank(rank int) Rank {
	if rank == 14 {
		return Ace
	}
	return Rank(rank)
}
