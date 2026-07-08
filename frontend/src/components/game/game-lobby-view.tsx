import {
  type CompletedGameSnapshot,
  type DealingChoiceRequest,
  type GameSnapshot,
  type PendingDealChoiceSnapshot,
  type PlayerSnapshot,
  type RoomSnapshot,
} from "#/components/game-websocket-provider";
import { ChevronDownIcon, Copy01Icon, CopyLinkIcon, Share08Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { DealChoicePanel } from "#/components/game/deal-choice-panel";
import { PlayerEmotePicker } from "#/components/game/player-emotes";
import { PlayerStrip } from "#/components/game/player-strip";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "#/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "#/components/ui/dropdown-menu";
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
  roomActions: RoomActions;
  dealChoice: DealChoiceState;
  onRoomCodeChange: (roomCode: string) => void;
  onCreateRoom: () => void;
  onJoinRoom: (roomCode: string) => void;
  onStartGame: () => void;
  onChooseDealing: (choice: DealingChoiceRequest | string) => void;
  onLeaveRoom: () => void;
  onSendEmote: (emoji: string) => void;
  onCopyRoomCode: AsyncAction;
  onCopyRoomLink: AsyncAction;
};

export function GameLobbyView({
  room,
  game,
  completedGame,
  players,
  roomCode,
  roomActions,
  dealChoice,
  onRoomCodeChange,
  onCreateRoom,
  onJoinRoom,
  onStartGame,
  onChooseDealing,
  onLeaveRoom,
  onSendEmote,
  onCopyRoomCode,
  onCopyRoomLink,
}: GameLobbyViewProps) {
  const { canCreateRoom, canJoinRoom, canLeaveRoom, canStartGame } = roomActions;
  const { pendingDealChoice, dealChooserName, isDealChooser } = dealChoice;
  const victorPlayerId = completedGame?.game.players[completedGame.game.roundWinnerIndex]?.playerId;
  const victor = players.find((player) => player.playerId === victorPlayerId) ?? null;
  const connectedCount = players.filter((player) => player.connected).length;

  return (
    <div className="mx-auto my-auto grid w-full max-w-5xl gap-4 lg:grid-cols-[minmax(0,1fr)_22rem]">
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
                  <DropdownMenu>
                    <DropdownMenuTrigger render={<Button type="button" variant="outline" />}>
                      <HugeiconsIcon icon={Share08Icon} strokeWidth={2} data-icon="inline-start" />
                      Share
                      <HugeiconsIcon
                        icon={ChevronDownIcon}
                        strokeWidth={2}
                        data-icon="inline-end"
                      />
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end" className="w-48">
                      <DropdownMenuItem onClick={() => void onCopyRoomCode()}>
                        <HugeiconsIcon icon={Copy01Icon} strokeWidth={2} />
                        Copy code
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => void onCopyRoomLink()}>
                        <HugeiconsIcon icon={CopyLinkIcon} strokeWidth={2} />
                        Copy link
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-2">
                <Button type="button" onClick={onStartGame} disabled={!canStartGame}>
                  Start game
                </Button>
                <Button
                  type="button"
                  variant="destructive"
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
        </CardContent>
      </Card>

      <Card className="border border-border/70 shadow-sm">
        <CardHeader>
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <CardTitle>Players</CardTitle>
              <CardDescription>{players.length ? "Seats" : "No room yet"}</CardDescription>
            </div>
            {players.length ? (
              <div className="flex items-center gap-2">
                <PlayerEmotePicker onSendEmote={onSendEmote} />
                <Badge variant="outline">{players.length} seated</Badge>
              </div>
            ) : null}
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
