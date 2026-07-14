import { UndoIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { type GameSnapshot, type PlayerSnapshot } from "#/components/game-websocket-provider";
import { PlayerEmotePicker } from "#/components/game/player-emotes";
import { PlayerStrip } from "#/components/game/player-strip";
import { AnimatedNumber } from "#/components/ui/animated-number";
import { Button } from "#/components/ui/button";
import { Card, CardAction, CardContent, CardHeader, CardTitle } from "#/components/ui/card";
import { Badge } from "../ui/badge";
import { m } from "#/paraglide/messages.js";

export function GameBoardPlayers({
  players,
  game,
  connectedPlayers,
  hasDraftedCompositions,
  showTurnIndicator = true,
  onResetDraftCompositions,
  onSendEmote,
}: {
  players: PlayerSnapshot[];
  game: GameSnapshot | null;
  connectedPlayers: number;
  hasDraftedCompositions: boolean;
  showTurnIndicator?: boolean;
  onResetDraftCompositions: () => void;
  onSendEmote: (emoji: string) => void;
}) {
  const activePlayerCount = players.filter((player) => !player.forfeited).length;
  return (
    <Card size="sm" className="min-w-0 grow overflow-y-auto">
      <CardHeader>
        <CardTitle>{m.players()}</CardTitle>
        <CardAction>
          <div className="flex items-center gap-2">
            <PlayerEmotePicker onSendEmote={onSendEmote} />
            <Badge variant="outline">
              <AnimatedNumber value={connectedPlayers} />/
              <AnimatedNumber value={activePlayerCount} /> {m.online()}
            </Badge>
          </div>
        </CardAction>
      </CardHeader>
      <CardContent className="flex h-full min-w-0 flex-col gap-3">
        <PlayerStrip
          players={players}
          game={game}
          showHostBadges={false}
          showTurnIndicator={showTurnIndicator}
        />

        {hasDraftedCompositions ? (
          <Button
            type="button"
            variant="ghost"
            onClick={onResetDraftCompositions}
            className="mt-auto w-full"
            size="lg"
          >
            <HugeiconsIcon icon={UndoIcon} strokeWidth={2} data-icon="inline-start" />
            {m.reset()}
          </Button>
        ) : null}
      </CardContent>
    </Card>
  );
}
