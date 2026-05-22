import { type GameSnapshot } from "#/components/game-websocket-provider";
import { formatLabel } from "#/components/game/game-view-utils";
import { Badge } from "#/components/ui/badge";
import { Card, CardContent } from "#/components/ui/card";

export function GameBoardHeader({
  connectionStatus,
  phase,
  roomCode,
  connectedPlayers,
  playerCount,
  isLobbyPhase,
  isMyTurn,
  turnPlayerName,
  game,
}: {
  connectionStatus: "idle" | "disconnected" | "connecting" | "connected";
  phase: string;
  roomCode?: string;
  connectedPlayers: number;
  playerCount: number;
  isLobbyPhase: boolean;
  isMyTurn: boolean;
  turnPlayerName: string;
  game: GameSnapshot | null;
}) {
  return (
    <Card size="sm" className="shadow-sm">
      <CardContent className="flex flex-col gap-3 py-0 lg:flex-row lg:items-center lg:justify-between">
        <div className="min-w-0">
          <p className="text-xs uppercase text-muted-foreground">
            {isLobbyPhase
              ? "Room status"
              : phase === "round_over"
                ? "Round complete"
                : phase === "game_over"
                  ? "Game complete"
                  : isMyTurn
                    ? "Your turn"
                    : `${turnPlayerName}'s turn`}
          </p>
          <p className="truncate text-sm font-medium">
            {isLobbyPhase
              ? roomCode
                ? `Room ${roomCode} is ready`
                : "Create or join a room"
              : phase === "round_over"
                ? `Round ${game?.round ?? ""} results are ready`
                : phase === "game_over"
                  ? "The game has ended"
                  : `${turnPlayerName} is playing ${formatLabel(phase).toLowerCase()}`}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Badge
            variant={
              connectionStatus === "connected"
                ? "default"
                : connectionStatus === "disconnected"
                  ? "destructive"
                  : "outline"
            }
          >
            {formatLabel(connectionStatus)}
          </Badge>
          <Badge variant="secondary">{formatLabel(phase)}</Badge>
          <Badge variant="outline">Room {roomCode ?? "None"}</Badge>
          {!isLobbyPhase && game ? (
            <>
              <Badge variant={isMyTurn ? "default" : "outline"}>Turn {game.turn.number}</Badge>
              <Badge variant="outline">Round {game.round}</Badge>
              <Badge variant="outline">{game.turn.hasDrawn ? "Drawn" : "Need draw"}</Badge>
              {game.turn.mustUseDiscardDraw ? (
                <Badge variant="destructive">Must use discard</Badge>
              ) : null}
            </>
          ) : null}
          <Badge variant="outline">
            {connectedPlayers}/{playerCount || 0} online
          </Badge>
        </div>
      </CardContent>
    </Card>
  );
}
