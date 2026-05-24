import { UndoIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { type GameSnapshot, type PlayerSnapshot } from "#/components/game-websocket-provider";
import { PlayerStrip } from "#/components/game/player-strip";
import { Button } from "#/components/ui/button";
import { Card, CardAction, CardContent, CardHeader, CardTitle } from "#/components/ui/card";
import { Badge } from "../ui/badge";

export function GameBoardPlayers({
  players,
  game,
  connectedPlayers,
  hasDraftedCompositions,
  onResetDraftCompositions,
}: {
  players: PlayerSnapshot[];
  game: GameSnapshot | null;
  connectedPlayers: number;
  hasDraftedCompositions: boolean;
  onResetDraftCompositions: () => void;
}) {
  return (
    <Card size="sm" className="grow overflow-y-auto">
      <CardHeader>
        <CardTitle>Players</CardTitle>
        <CardAction>
          <Badge variant="outline">
            {connectedPlayers}/{players.length || 0} online
          </Badge>
        </CardAction>
      </CardHeader>
      <CardContent className="flex h-full flex-col gap-3">
        <PlayerStrip players={players} game={game} />

        {hasDraftedCompositions ? (
          <Button
            type="button"
            variant="ghost"
            onClick={onResetDraftCompositions}
            className="mt-auto w-full"
            size="lg"
          >
            <HugeiconsIcon icon={UndoIcon} strokeWidth={2} data-icon="inline-start" />
            Reset
          </Button>
        ) : null}
      </CardContent>
    </Card>
  );
}
