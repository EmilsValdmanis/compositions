import {
  type CompletedGameSnapshot,
  type GameSnapshot,
  type PendingDealChoiceSnapshot,
  type PlayerSnapshot,
  type RoomSnapshot,
} from "#/components/game-websocket-provider";
import { PlayerStrip } from "#/components/game/player-strip";
import { compactId } from "#/components/game/game-view-helpers";
import { Button } from "#/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "#/components/ui/card";
import { Input } from "#/components/ui/input";

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
  onChooseDealing: (dealType: string) => void;
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

  return (
    <div className="grid gap-4 lg:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)]">
      <Card>
        <CardHeader>
          <CardTitle>Lobby</CardTitle>
          <CardDescription>{room ? "Room is open" : "Create or join a room"}</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4">
          {room ? (
            <div className="grid gap-4">
              <div className="grid gap-4 rounded-3xl border border-border/70 bg-muted/20 p-4">
                <div className="grid gap-1">
                  <p className="text-xs uppercase tracking-[0.2em] text-muted-foreground">
                    Room code
                  </p>
                  <div className="flex flex-wrap items-end justify-between gap-3">
                    <div>
                      <p className="text-3xl font-semibold tracking-tight">{room.code}</p>
                      <p className="text-sm text-muted-foreground">
                        Copy the join link to drop players straight into this lobby.
                      </p>
                    </div>
                    <Button type="button" variant="outline" onClick={() => void onCopyRoomCode()}>
                      Copy code
                    </Button>
                  </div>
                </div>

                <div className="rounded-3xl border border-border/70 bg-background/80 p-3">
                  <div className="flex flex-col gap-2 sm:flex-row">
                    <Input
                      value={roomLink ?? room.code}
                      readOnly
                      aria-label="Room join link"
                      className="font-mono text-xs"
                    />
                    <Button
                      type="button"
                      className="sm:ml-auto"
                      onClick={() => void onCopyRoomLink()}
                    >
                      Copy join link
                    </Button>
                  </div>
                </div>
              </div>
              <div className="flex flex-wrap gap-2">
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
                <div className="grid gap-3 rounded-3xl border border-border/70 bg-muted/20 p-4">
                  <div className="grid gap-1">
                    <p className="text-sm font-medium">Choose dealing type</p>
                    <p className="text-sm text-muted-foreground">
                      {isDealChooser
                        ? "You need to choose how this round will be dealt."
                        : `${dealChooserName ?? "A player"} needs to choose how this round will be dealt.`}
                    </p>
                  </div>
                  <div className="flex flex-col gap-2 sm:flex-row">
                    <Button
                      type="button"
                      onClick={() => onChooseDealing("round_robin")}
                      disabled={!isDealChooser}
                    >
                      Round robin
                    </Button>
                    <Button type="button" variant="outline" disabled>
                      Tap dealing coming later
                    </Button>
                  </div>
                </div>
              ) : null}
              {completedGame ? (
                <div className="grid gap-2 rounded-3xl border border-border/70 bg-muted/20 p-4">
                  <p className="text-sm font-medium">Last game winner</p>
                  <p className="text-sm text-muted-foreground">
                    {victor?.name ?? "A player"} won the game in round {completedGame.game.round}.
                  </p>
                </div>
              ) : null}
            </div>
          ) : (
            <div className="grid gap-3">
              <Button type="button" onClick={onCreateRoom} disabled={!canCreateRoom}>
                Create room
              </Button>
              <div className="flex flex-col gap-2 sm:flex-row">
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

          <div className="grid gap-2 rounded-3xl border border-border/70 bg-muted/20 p-4 text-sm">
            <div className="flex justify-between gap-3">
              <span className="text-muted-foreground">Player ID</span>
              <span className="font-medium">{compactId(playerId)}</span>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Players</CardTitle>
          <CardDescription>
            {players.length ? "Seats and connection state" : "No room yet"}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {players.length ? (
            <PlayerStrip players={players} game={game} />
          ) : (
            <div className="rounded-3xl border border-dashed border-border/70 p-6 text-sm text-muted-foreground">
              Connect first, then create or join.
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
