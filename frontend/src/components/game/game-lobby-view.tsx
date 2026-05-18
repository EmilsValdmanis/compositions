import {
  type GameSnapshot,
  type PendingDealChoiceSnapshot,
  type PlayerSnapshot,
  type RoomSnapshot,
} from "#/components/game-websocket-provider";
import { PlayerStrip } from "#/components/game/player-strip";
import { compactId } from "#/components/game/game-view-utils";
import { Button } from "#/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "#/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "#/components/ui/dropdown-menu";
import { Input } from "#/components/ui/input";

type AsyncAction = () => Promise<void> | void;

export function GameLobbyView({
  room,
  game,
  players,
  roomCode,
  playerId,
  sessionId,
  canCreateRoom,
  canJoinRoom,
  canLeaveRoom,
  canStartGame,
  pendingDealChoice,
  dealChooserName,
  isDealChooser,
  onRoomCodeChange,
  onCreateRoom,
  onJoinRoom,
  onStartGame,
  onChooseDealing,
  onLeaveRoom,
  onCopyRoomCode,
  onCopyRoomLink,
  onShareRoom,
}: {
  room: RoomSnapshot | null;
  game: GameSnapshot | null;
  players: PlayerSnapshot[];
  roomCode: string;
  playerId: string;
  sessionId: string;
  canCreateRoom: boolean;
  canJoinRoom: boolean;
  canLeaveRoom: boolean;
  canStartGame: boolean;
  pendingDealChoice: PendingDealChoiceSnapshot | null;
  dealChooserName: string | null;
  isDealChooser: boolean;
  onRoomCodeChange: (roomCode: string) => void;
  onCreateRoom: () => void;
  onJoinRoom: (roomCode: string) => void;
  onStartGame: () => void;
  onChooseDealing: (dealType: string) => void;
  onLeaveRoom: () => void;
  onCopyRoomCode: AsyncAction;
  onCopyRoomLink: AsyncAction;
  onShareRoom: AsyncAction;
}) {
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
              <div className="rounded-3xl border border-border/70 bg-muted/20 p-4">
                <p className="text-xs uppercase text-muted-foreground">Room code</p>
                <div className="mt-2 flex flex-wrap items-center justify-between gap-3">
                  <p className="text-3xl font-semibold tracking-tight">{room.code}</p>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button type="button" variant="outline">
                        Room actions
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem onClick={() => void onCopyRoomCode()}>
                        Copy code
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => void onCopyRoomLink()}>
                        Copy link
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => void onShareRoom()}>
                        Share link
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem onClick={onLeaveRoom} disabled={!canLeaveRoom}>
                        Leave room
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
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
                  onClick={() => onJoinRoom(roomCode.trim().toUpperCase())}
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
            <div className="flex justify-between gap-3">
              <span className="text-muted-foreground">Session ID</span>
              <span className="font-medium">{compactId(sessionId)}</span>
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
