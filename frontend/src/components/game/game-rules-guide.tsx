import { type ComponentProps, type ReactNode } from "react";
import { ArrowRight01Icon, PaintBoardIcon, Target02Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";

import { type CardSnapshot } from "#/components/game-websocket-provider";
import { GameCard } from "#/components/game/game-card";
import { Badge } from "#/components/ui/badge";
import { Card, CardContent, CardHeader } from "#/components/ui/card";
import { Item, ItemContent, ItemGroup, ItemMedia } from "#/components/ui/item";
import { H4, Text } from "#/components/typography";
import { cn } from "#/lib/utils";

type GameRulesGuideProps = {
  className?: string;
};

type RuleSectionProps = {
  eyebrow: string;
  title: string;
  children: ReactNode;
};

function card(rank: number, suit: number): CardSnapshot {
  return { rank, suit };
}

function joker(): CardSnapshot {
  return { isJoker: true };
}

function RuleSection({ eyebrow, title, children }: RuleSectionProps) {
  return (
    <Card size="sm" className="shadow-none">
      <CardHeader>
        <Text as="p" variant="eyebrow-compact">
          {eyebrow}
        </Text>
        <Text as="h3" variant="body-strong">
          {title}
        </Text>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">{children}</CardContent>
    </Card>
  );
}

function RuleTitle({ children }: { children: ReactNode }) {
  return (
    <Text as="div" variant="label" data-slot="item-title">
      {children}
    </Text>
  );
}

function RuleDescription({ children }: { children: ReactNode }) {
  return (
    <Text
      as="p"
      variant="caption"
      data-slot="item-description"
      className="line-clamp-none text-left"
    >
      {children}
    </Text>
  );
}

function RuleList({ items }: { items: string[] }) {
  return (
    <ItemGroup className="gap-2">
      {items.map((item) => (
        <Item key={item} variant="muted" size="xs">
          <ItemMedia className="pt-[0.3rem]">
            <span className="size-1.5 rounded-full bg-primary" />
          </ItemMedia>
          <ItemContent>
            <RuleDescription>{item}</RuleDescription>
          </ItemContent>
        </Item>
      ))}
    </ItemGroup>
  );
}

function CardLine({
  cards,
  label,
  wrap = false,
}: {
  cards: CardSnapshot[];
  label?: string;
  wrap?: boolean;
}) {
  return (
    <div className="flex flex-col gap-2">
      {label ? (
        <Text as="p" variant="label" className="text-muted-foreground">
          {label}
        </Text>
      ) : null}
      <div
        className={cn(
          "flex min-w-0 items-center gap-1.5 overflow-x-auto pb-1",
          wrap ? "flex-wrap overflow-x-visible pb-0" : null,
        )}
      >
        {cards.map((exampleCard, index) => (
          <GameCard
            key={`${
              exampleCard.isJoker ? "joker" : `${exampleCard.rank}-${exampleCard.suit}`
            }-${index}`}
            card={exampleCard}
            size="compact"
          />
        ))}
      </div>
    </div>
  );
}

function FlowStep({ children }: { children: ReactNode }) {
  return (
    <Item variant="outline" size="xs" className="flex-1 justify-center">
      <RuleTitle>{children}</RuleTitle>
    </Item>
  );
}

function TurnFlow() {
  return (
    <div className="flex items-center gap-1">
      <FlowStep>Draw</FlowStep>
      <HugeiconsIcon icon={ArrowRight01Icon} className="shrink-0" />
      <FlowStep>Build</FlowStep>
      <HugeiconsIcon icon={ArrowRight01Icon} className="shrink-0" />
      <FlowStep>Discard</FlowStep>
    </div>
  );
}

function ScorePills() {
  const scores = ["2-10 = face value", "J Q K = 10", "Ace = 1 or 10", "Joker = 20"];

  return (
    <div className="flex flex-wrap gap-2">
      {scores.map((score) => (
        <Badge key={score} variant="secondary">
          {score}
        </Badge>
      ))}
    </div>
  );
}

function ReclaimExample() {
  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-end gap-3 overflow-x-auto pb-1">
        <CardLine cards={[card(3, 3), joker(), card(5, 3)]} label="Run on table: joker is 4" />
        <HugeiconsIcon icon={ArrowRight01Icon} className="mb-6 shrink-0" />
        <CardLine cards={[card(4, 3)]} label="Replace with exact card" />
      </div>
      <RuleDescription>
        The 4 replaces the joker, then the joker returns to your hand for this turn.
      </RuleDescription>
    </div>
  );
}

function OpeningExample() {
  return (
    <div className="flex flex-col gap-2">
      <Text as="p" variant="label" className="text-muted-foreground">
        Example opening: Kings set 30 + spade run 15 = 45
      </Text>
      <div className="flex flex-wrap items-center gap-4">
        <CardLine cards={[card(13, 0), card(13, 1), card(13, 2)]} />
        <Badge variant="outline">+</Badge>
        <CardLine cards={[card(4, 3), card(5, 3), card(6, 3)]} />
      </div>
    </div>
  );
}

function SpecialWinTile({
  icon,
  title,
  description,
}: {
  icon: ComponentProps<typeof HugeiconsIcon>["icon"];
  title: string;
  description: string;
}) {
  return (
    <Item variant="outline" size="sm" className="items-start">
      <ItemMedia>
        <span className="grid size-7 place-items-center rounded-full bg-primary/10 text-primary">
          <HugeiconsIcon icon={icon} />
        </span>
      </ItemMedia>
      <ItemContent>
        <RuleTitle>{title}</RuleTitle>
        <RuleDescription>{description}</RuleDescription>
      </ItemContent>
    </Item>
  );
}

export function GameRulesGuide({ className }: GameRulesGuideProps) {
  return (
    <div className={cn("flex min-h-0 flex-col gap-4", className)}>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <Text as="p" variant="eyebrow-compact">
            Quick rules
          </Text>
          <H4>Compositions</H4>
        </div>
        <Badge variant="outline">2 decks</Badge>
      </div>

      <div className="flex flex-col gap-3">
        <RuleSection eyebrow="Goal" title="Empty your hand by discarding your last card">
          <RuleList
            items={[
              "You start each round with 12 random cards.",
              "Win the round by playing down cards, then discarding the final card from your hand.",
              "Keep one card for the final discard; you cannot go out by placing every card on the table.",
              "Cards left in other players' hands become points against them.",
            ]}
          />
        </RuleSection>

        <RuleSection eyebrow="Turn" title="Draw, compose, discard">
          <TurnFlow />
          <RuleList
            items={[
              "Draw from the deck, or take the top discard only if you can use it immediately.",
              "Playing compositions is optional after you have drawn.",
              "A turn ends when you discard one card.",
            ]}
          />
        </RuleSection>

        <RuleSection eyebrow="Compositions" title="Two valid shapes">
          <div className="flex flex-wrap gap-3">
            <Item variant="outline" size="sm" className="w-fit min-w-52 items-start">
              <ItemContent className="gap-2">
                <RuleTitle>Set: same rank</RuleTitle>
                <CardLine cards={[card(7, 0), card(7, 1), card(7, 2)]} />
                <RuleDescription>3+ cards, different suits.</RuleDescription>
              </ItemContent>
            </Item>
            <Item variant="outline" size="sm" className="w-fit min-w-60 items-start">
              <ItemContent className="gap-2">
                <RuleTitle>Run</RuleTitle>
                <CardLine cards={[card(5, 3), card(6, 3), card(7, 3), card(8, 3)]} />
                <RuleDescription>3+ cards in order, same suit.</RuleDescription>
              </ItemContent>
            </Item>
          </div>
        </RuleSection>

        <RuleSection eyebrow="Opening" title="Your first table play must reach 40+ points">
          <div className="flex flex-col gap-3">
            <OpeningExample />
            <RuleList
              items={[
                "Before opening, you cannot play random additions by themselves.",
                "Your opening turn must include at least one new composition from your hand.",
                "Jokers in hand can help reach 40 by taking the value they represent.",
              ]}
            />
          </div>
        </RuleSection>

        <RuleSection eyebrow="Jokers" title="Replace the exact card to reclaim a joker">
          <div className="flex flex-col gap-3">
            <ReclaimExample />
            <RuleList
              items={[
                "A joker can stand for any card in a set or run.",
                "To take it back, place the exact card it currently represents.",
                "In sets, the joker must be narrowed to one missing suit before it can be reclaimed.",
              ]}
            />
          </div>
        </RuleSection>

        <RuleSection eyebrow="Complete" title="Finished compositions leave the table">
          <div className="flex flex-col gap-3">
            <CardLine
              cards={[card(1, 0), card(1, 1), card(1, 2), card(1, 3)]}
              label="Complete Ace set: all four suits"
            />
            <RuleList
              items={[
                "A complete composition is moved to the discard pile before the final discard.",
                "A same-suit run is complete only when it covers Ace low through King plus the second Ace high.",
                "Completed cards leave the table, then the current player discards normally.",
              ]}
            />
          </div>
        </RuleSection>

        <RuleSection eyebrow="Scores" title="There are two scoring contexts">
          <ScorePills />
          <ItemGroup className="grid gap-3 sm:grid-cols-2">
            <Item variant="outline" size="sm" className="items-start">
              <ItemContent>
                <RuleTitle>Opening points</RuleTitle>
                <RuleDescription>
                  Count the value of cards you place in compositions. Aces and jokers use the value
                  they represent in that composition.
                </RuleDescription>
              </ItemContent>
            </Item>
            <Item variant="outline" size="sm" className="items-start">
              <ItemContent>
                <RuleTitle>End-round points</RuleTitle>
                <RuleDescription>
                  Only cards left in hand count. Jokers are 20. Aces are usually 10, except a final
                  lone Ace can count as 1.
                </RuleDescription>
              </ItemContent>
            </Item>
          </ItemGroup>
          <RuleList
            items={[
              "Low total score is good.",
              "Players over 100 are adjusted unless everyone else is also over 100.",
              "You win the game by forcing all other players over 100.",
            ]}
          />
        </RuleSection>

        <RuleSection eyebrow="Special wins" title="Rare hands can end it immediately">
          <ItemGroup className="grid gap-3 sm:grid-cols-2">
            <SpecialWinTile
              icon={PaintBoardIcon}
              title="Same suit collection"
              description="Collect all 12 cards of one suit for an immediate win."
            />
            <SpecialWinTile
              icon={Target02Icon}
              title="Six identical pairs"
              description="Form 6 identical pairs for a special win condition."
            />
          </ItemGroup>
        </RuleSection>
      </div>
    </div>
  );
}
