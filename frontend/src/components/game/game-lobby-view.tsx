import {
  type CompletedGameSnapshot,
  type DealingChoiceRequest,
  type GameSnapshot,
  type PendingDealChoiceSnapshot,
  type PlayerSnapshot,
  type RoomSnapshot,
} from "#/components/game-websocket-provider";
import { DealChoicePanel } from "#/components/game/deal-choice-panel";
import { PlayerStrip } from "#/components/game/player-strip";
import { compactId } from "#/components/game/game-view-helpers";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "#/components/ui/card";
import { Input } from "#/components/ui/input";
import { Separator } from "#/components/ui/separator";

type AsyncAction = () => Promise<void> | void;

type RoomActions = {
  canCreateRoom: boolean;
  canJoinRoom: boolean;
  canLeaveRoom: boolean;
  canStartGame: boolean;
};

type DealChoiceState = {
  pendingDealChoice: PendingDealChoiceSnapshot | null;
  dealChooserName: string | null;
  isDealChooser: boolean;
};

type GameLobbyViewProps = {
  room: RoomSnapshot | null;
  game: GameSnapshot | null;
  completedGame: CompletedGameSnapshot | null;
  players: PlayerSnapshot[];
  roomCode: string;
  playerId: string;
  roomActions: RoomActions;
  dealChoice: DealChoiceState;
  onRoomCodeChange: (roomCode: string) => void;
  onCreateRoom: () => void;
  onJoinRoom: (roomCode: string) => void;
  onStartGame: () => void;
  onChooseDealing: (choice: DealingChoiceRequest | string) => void;
  onLeaveRoom: () => void;
  onCopyRoomCode: AsyncAction;
  onCopyRoomLink: AsyncAction;
  roomLink: string | null;
};

export function GameLobbyView({
  room,
  game,
  completedGame,
  players,
  roomCode,
  playerId,
  roomActions,
  dealChoice,
  onRoomCodeChange,
  onCreateRoom,
  onJoinRoom,
  onStartGame,
  onChooseDealing,
  onLeaveRoom,
  onCopyRoomCode,
  onCopyRoomLink,
  roomLink,
}: GameLobbyViewProps) {
  const { canCreateRoom, canJoinRoom, canLeaveRoom, canStartGame } = roomActions;
  const { pendingDealChoice, dealChooserName, isDealChooser } = dealChoice;
  const victorPlayerId = completedGame?.game.players[completedGame.game.roundWinnerIndex]?.playerId;
  const victor = players.find((player) => player.playerId === victorPlayerId) ?? null;
  const connectedCount = players.filter((player) => player.connected).length;

  return (
    <div className="grid gap-4 lg:grid-cols-[minmax(320px,0.85fr)_minmax(0,1.15fr)]">
      <Card className="border border-border/70 shadow-sm">
        <CardHeader>
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <CardTitle>Lobby</CardTitle>
              <CardDescription>{room ? "Ready room" : "Create or join"}</CardDescription>
            </div>
            <Badge variant={room ? "secondary" : "outline"}>
              {room ? `${connectedCount}/${players.length} online` : "Offline"}
            </Badge>
          </div>
        </CardHeader>
        <CardContent className="grid gap-4">
          {room ? (
            <div className="grid gap-4">
              <div className="grid gap-4 rounded-2xl border border-border/70 bg-muted/20 p-4">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">
                      Room code
                    </p>
                    <p className="text-4xl font-semibold tracking-tight">{room.code}</p>
                  </div>
                  <Button type="button" variant="outline" onClick={() => void onCopyRoomCode()}>
                    Copy code
                  </Button>
                </div>

                <div className="flex flex-col gap-2 sm:flex-row">
                  <Input
                    value={roomLink ?? room.code}
                    readOnly
                    aria-label="Room join link"
                    className="h-10 font-mono text-xs"
                  />
                  <Button
                    type="button"
                    className="sm:ml-auto"
                    onClick={() => void onCopyRoomLink()}
                  >
                    Copy link
                  </Button>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-2">
                <Button type="button" onClick={onStartGame} disabled={!canStartGame}>
                  Start game
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  onClick={onLeaveRoom}
                  disabled={!canLeaveRoom}
                >
                  Leave room
                </Button>
              </div>
              {pendingDealChoice ? (
                <DealChoicePanel
                  players={players}
                  pendingDealChoice={pendingDealChoice}
                  dealChooserName={dealChooserName}
                  isDealChooser={isDealChooser}
                  onChooseDealing={onChooseDealing}
                />
              ) : null}
              {completedGame ? (
                <div className="grid gap-1 rounded-2xl border border-border/70 bg-muted/20 p-4">
                  <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">
                    Last winner
                  </p>
                  <p className="text-sm text-muted-foreground">
                    <span className="font-medium text-foreground">
                      {victor?.name ?? "A player"}
                    </span>{" "}
                    in round {completedGame.game.round}
                  </p>
                </div>
              ) : null}
            </div>
          ) : (
            <div className="grid gap-4">
              <Button type="button" onClick={onCreateRoom} disabled={!canCreateRoom}>
                Create room
              </Button>
              <div className="grid gap-2">
                <Separator />
                <Input
                  value={roomCode}
                  onChange={(event) => onRoomCodeChange(event.target.value.toUpperCase())}
                  placeholder="Room code"
                  maxLength={6}
                />
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => onJoinRoom(roomCode.trim())}
                  disabled={!canJoinRoom}
                >
                  Join
                </Button>
              </div>
            </div>
          )}

          <div className="grid gap-2 rounded-2xl border border-border/70 bg-background p-3 text-sm">
            <div className="flex justify-between gap-3">
              <span className="text-muted-foreground">Player ID</span>
              <span className="font-medium">{compactId(playerId)}</span>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card className="border border-border/70 shadow-sm">
        <CardHeader>
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <CardTitle>Players</CardTitle>
              <CardDescription>{players.length ? "Seats" : "No room yet"}</CardDescription>
            </div>
            {players.length ? <Badge variant="outline">{players.length} seated</Badge> : null}
          </div>
        </CardHeader>
        <CardContent>
          {players.length ? (
            <PlayerStrip players={players} game={game} />
          ) : (
            <div className="rounded-2xl border border-dashed border-border/70 p-6 text-sm text-muted-foreground">
              Create or join a room.
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
