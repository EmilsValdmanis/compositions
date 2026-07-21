import { type ComponentProps, type ReactNode } from "react";
import { ArrowRight01Icon, PaintBoardIcon, Target02Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";

import { type CardSnapshot } from "#/components/game-websocket-provider";
import { GameCard } from "#/components/game/game-card";
import { Badge } from "#/components/ui/badge";
import { Card, CardContent, CardHeader } from "#/components/ui/card";
import { Item, ItemContent, ItemGroup, ItemMedia } from "#/components/ui/item";
import { Caption, H4, H6, P } from "#/components/typography";
import { cn } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";
import { useIsMobile } from "#/hooks/use-mobile";

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
  const isMobile = useIsMobile();
  return (
    <Card size={isMobile ? "sm" : "default"} className="shadow-none">
      <CardHeader>
        <Caption className="uppercase">{eyebrow}</Caption>
        <H6>{title}</H6>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">{children}</CardContent>
    </Card>
  );
}

function RuleTitle({ children }: { children: ReactNode }) {
  return (
    <P size="xs" className="font-medium" data-slot="item-title">
      {children}
    </P>
  );
}

function RuleDescription({ children }: { children: ReactNode }) {
  return (
    <Caption data-slot="item-description" className="line-clamp-none text-left">
      {children}
    </Caption>
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
        <P size="xs" className="font-medium text-muted-foreground">
          {label}
        </P>
      ) : null}
      <div
        className={cn(
          "flex min-w-0 items-center gap-1.5 overflow-x-auto pb-1",
          wrap ? "flex-wrap overflow-x-visible pb-0" : null,
        )}
      >
        {cards.map((exampleCard) => (
          <GameCard
            key={
              exampleCard.isJoker
                ? "joker"
                : `${exampleCard.rank ?? "unknown"}-${exampleCard.suit ?? "unknown"}`
            }
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
      <FlowStep>{m.draw()}</FlowStep>
      <HugeiconsIcon icon={ArrowRight01Icon} className="shrink-0" />
      <FlowStep>{m.build()}</FlowStep>
      <HugeiconsIcon icon={ArrowRight01Icon} className="shrink-0" />
      <FlowStep>{m.discard()}</FlowStep>
    </div>
  );
}

function ScorePills() {
  const scores = [m.score_face_value(), m.score_faces(), m.score_ace(), m.score_joker()];

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
        <CardLine cards={[card(3, 3), joker(), card(5, 3)]} label={m.reclaim_run_label()} />
        <HugeiconsIcon icon={ArrowRight01Icon} className="mb-6 shrink-0" />
        <CardLine cards={[card(4, 3)]} label={m.reclaim_exact_label()} />
      </div>
      <RuleDescription>{m.reclaim_description()}</RuleDescription>
    </div>
  );
}

function OpeningExample() {
  return (
    <div className="flex flex-col gap-2">
      <P size="xs" className="font-medium text-muted-foreground">
        {m.opening_example()}
      </P>
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
          <Caption className="uppercase">{m.quick_rules()}</Caption>
          <H4>{m.compositions()}</H4>
        </div>
        <Badge variant="outline">{m.two_decks()}</Badge>
      </div>

      <div className="flex flex-col gap-3">
        <RuleSection eyebrow={m.goal()} title={m.goal_title()}>
          <RuleList items={[m.goal_rule_1(), m.goal_rule_2(), m.goal_rule_3(), m.goal_rule_4()]} />
        </RuleSection>

        <RuleSection eyebrow={m.turn()} title={m.turn_title()}>
          <TurnFlow />
          <RuleList items={[m.turn_rule_1(), m.turn_rule_2(), m.turn_rule_3()]} />
        </RuleSection>

        <RuleSection eyebrow={m.compositions()} title={m.compositions_title()}>
          <div className="flex flex-wrap gap-3">
            <Item variant="outline" size="sm" className="w-fit min-w-52 items-start">
              <ItemContent className="gap-2">
                <RuleTitle>{m.set_same_rank()}</RuleTitle>
                <CardLine cards={[card(7, 0), card(7, 1), card(7, 2)]} />
                <RuleDescription>{m.set_description()}</RuleDescription>
              </ItemContent>
            </Item>
            <Item variant="outline" size="sm" className="w-fit min-w-60 items-start">
              <ItemContent className="gap-2">
                <RuleTitle>{m.run()}</RuleTitle>
                <CardLine cards={[card(5, 3), card(6, 3), card(7, 3), card(8, 3)]} />
                <RuleDescription>{m.run_description()}</RuleDescription>
              </ItemContent>
            </Item>
          </div>
        </RuleSection>

        <RuleSection eyebrow={m.opening()} title={m.opening_title()}>
          <div className="flex flex-col gap-3">
            <OpeningExample />
            <RuleList items={[m.opening_rule_1(), m.opening_rule_2(), m.opening_rule_3()]} />
          </div>
        </RuleSection>

        <RuleSection eyebrow={m.jokers()} title={m.jokers_title()}>
          <div className="flex flex-col gap-3">
            <ReclaimExample />
            <RuleList items={[m.joker_rule_1(), m.joker_rule_2(), m.joker_rule_3()]} />
          </div>
        </RuleSection>

        <RuleSection eyebrow={m.complete_section()} title={m.complete_title()}>
          <div className="flex flex-col gap-3">
            <CardLine
              cards={[card(1, 0), card(1, 1), card(1, 2), card(1, 3)]}
              label={m.complete_ace_set()}
            />
            <RuleList items={[m.complete_rule_1(), m.complete_rule_2(), m.complete_rule_3()]} />
          </div>
        </RuleSection>

        <RuleSection eyebrow={m.scores()} title={m.scores_title()}>
          <ScorePills />
          <ItemGroup className="grid gap-3 sm:grid-cols-2">
            <Item variant="outline" size="sm" className="items-start">
              <ItemContent>
                <RuleTitle>{m.opening_points()}</RuleTitle>
                <RuleDescription>{m.opening_points_description()}</RuleDescription>
              </ItemContent>
            </Item>
            <Item variant="outline" size="sm" className="items-start">
              <ItemContent>
                <RuleTitle>{m.end_round_points()}</RuleTitle>
                <RuleDescription>{m.end_round_points_description()}</RuleDescription>
              </ItemContent>
            </Item>
          </ItemGroup>
          <RuleList items={[m.score_rule_1(), m.score_rule_2(), m.score_rule_3()]} />
        </RuleSection>

        <RuleSection eyebrow={m.special_wins()} title={m.special_wins_title()}>
          <ItemGroup className="grid gap-3 sm:grid-cols-2">
            <SpecialWinTile
              icon={PaintBoardIcon}
              title={m.same_suit_collection()}
              description={m.same_suit_description()}
            />
            <SpecialWinTile
              icon={Target02Icon}
              title={m.six_pairs()}
              description={m.six_pairs_description()}
            />
          </ItemGroup>
        </RuleSection>
      </div>
    </div>
  );
}
