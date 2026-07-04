import { useEffect, useState } from "react";
import { closestCenter, DndContext, type DragEndEvent } from "@dnd-kit/core";
import {
  arrayMove,
  SortableContext,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import {
  type CompletedGameSnapshot,
  type DealingChoiceRequest,
  type GameSnapshot,
  type PendingDealChoiceSnapshot,
  type PlayerSnapshot,
  type RoomSnapshot,
} from "#/components/game-websocket-provider";
import { PlayerStrip } from "#/components/game/player-strip";
import { compactId } from "#/components/game/game-view-helpers";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "#/components/ui/card";
import { Input } from "#/components/ui/input";
import { Label } from "#/components/ui/label";
import { Separator } from "#/components/ui/separator";
import { cn } from "#/lib/utils";

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

const GAME_DECK_CARD_COUNT = 108;

type DealMode = "cut" | "tap";

function defaultTapOrder(players: PlayerSnapshot[], pendingDealChoice: PendingDealChoiceSnapshot) {
  return players.map((_, offset) => (pendingDealChoice.dealerIndex + offset + 1) % players.length);
}

function dealPlayerId(playerIndex: number) {
  return `deal-player-${playerIndex}`;
}

function playerIndexFromDealId(id: string) {
  return Number(id.replace("deal-player-", ""));
}

function clampCutSize(rawCutSize: string, maxCutSize: number) {
  const parsed = Number.parseInt(rawCutSize, 10);

  if (!Number.isFinite(parsed)) {
    return 0;
  }

  return Math.min(Math.max(parsed, 0), maxCutSize);
}

function CutDeckControl({
  cutSize,
  clampedCutSize,
  maxCutSize,
  onCutSizeChange,
}: {
  cutSize: string;
  clampedCutSize: number;
  maxCutSize: number;
  onCutSizeChange: (cutSize: string) => void;
}) {
  const cutRatio = maxCutSize > 0 ? clampedCutSize / maxCutSize : 0;
  const lift = Math.round(10 + cutRatio * 54);
  const slide = Math.round(4 + cutRatio * 28);
  const tilt = Math.round((cutRatio - 0.5) * 8);

  return (
    <div className="grid gap-3 rounded-lg border border-border/70 bg-background/80 p-3">
      <div className="flex items-center justify-between gap-3">
        <Label htmlFor="cut-size" className="text-xs font-medium uppercase tracking-[0.18em]">
          Cut size
        </Label>
        <Badge variant="secondary">{clampedCutSize} cards</Badge>
      </div>

      <div className="grid grid-cols-[minmax(0,1fr)_2.5rem] items-center gap-3">
        <div className="relative h-32 overflow-hidden rounded-lg border border-dashed border-border/70 bg-muted/20">
          <div className="absolute left-1/2 top-7 h-20 w-24 -translate-x-1/2">
            {[0, 1, 2, 3].map((index) => (
              <div
                key={index}
                className="absolute h-16 w-20 rounded-md border border-border bg-card shadow-sm"
                style={{
                  left: `${index * 2}px`,
                  top: `${index * 3}px`,
                }}
              />
            ))}
            <div
              className="absolute h-16 w-20 rounded-md border border-primary/50 bg-primary/10 shadow-md transition-transform duration-200 ease-out"
              style={{
                transform: `translate(${slide}px, -${lift}px) rotate(${tilt}deg)`,
              }}
            >
              <div className="m-2 h-2 rounded-full bg-primary/35" />
              <div className="mx-2 mt-7 h-2 rounded-full bg-primary/20" />
            </div>
          </div>
          <div className="absolute inset-x-4 bottom-3 h-px bg-border" />
          <p className="absolute bottom-4 left-4 text-xs text-muted-foreground">Cut packet</p>
        </div>

        <Input
          id="cut-size"
          type="range"
          min={0}
          max={maxCutSize}
          value={clampedCutSize}
          onChange={(event) => onCutSizeChange(event.target.value)}
          className="h-28 w-8 cursor-pointer p-0 accent-primary [direction:rtl] [writing-mode:vertical-lr]"
        />
      </div>

      <div className="grid grid-cols-[minmax(0,1fr)_5rem] items-center gap-2">
        <p className="text-xs text-muted-foreground">
          Main {maxCutSize - clampedCutSize} / packet {clampedCutSize}
        </p>
        <Input
          type="number"
          inputMode="numeric"
          min={0}
          max={maxCutSize}
          value={cutSize}
          onChange={(event) => onCutSizeChange(event.target.value)}
          aria-label="Exact cut size"
          className="h-9 text-right"
        />
      </div>
    </div>
  );
}

function SortableDealOrderPlayer({
  player,
  playerIndex,
  position,
}: {
  player: PlayerSnapshot;
  playerIndex: number;
  position: number;
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: dealPlayerId(playerIndex),
  });
  const style = {
    transform: transform
      ? `translate3d(${Math.round(transform.x)}px, ${Math.round(transform.y)}px, 0)`
      : undefined,
    transition,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={cn(
        "flex min-h-14 items-center gap-3 rounded-lg border border-border/70 bg-background px-3 py-2 text-sm shadow-sm",
        isDragging ? "opacity-60" : undefined,
      )}
      {...attributes}
      {...listeners}
    >
      <Badge variant="outline" className="w-9 justify-center">
        {position + 1}
      </Badge>
      <div className="min-w-0 flex-1">
        <p className="truncate font-medium">{player.name}</p>
        <p className="text-xs text-muted-foreground">Seat {player.seat + 1}</p>
      </div>
    </div>
  );
}

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
  const [dealMode, setDealMode] = useState<DealMode>("cut");
  const [cutSize, setCutSize] = useState("0");
  const [tapOrder, setTapOrder] = useState<number[]>([]);
  const dealChoiceKey = pendingDealChoice
    ? [
        pendingDealChoice.dealerIndex,
        pendingDealChoice.chooserIndex,
        players.map((player) => player.playerId).join(","),
      ].join(":")
    : null;
  const maxCutSize = Math.max(0, GAME_DECK_CARD_COUNT - players.length * 12);
  const clampedCutSize = clampCutSize(cutSize, maxCutSize);
  const dealerName =
    pendingDealChoice && players[pendingDealChoice.dealerIndex]
      ? players[pendingDealChoice.dealerIndex].name
      : null;
  const tapOrderIds = tapOrder.map(dealPlayerId);

  useEffect(() => {
    if (!pendingDealChoice) {
      setDealMode("cut");
      setCutSize("0");
      setTapOrder([]);
      return;
    }

    setDealMode("cut");
    setCutSize("0");
    setTapOrder(defaultTapOrder(players, pendingDealChoice));
  }, [dealChoiceKey, pendingDealChoice, players]);

  function handleTapOrderDragEnd(event: DragEndEvent) {
    const { active, over } = event;

    if (!over || active.id === over.id) {
      return;
    }

    const activeIndex = playerIndexFromDealId(String(active.id));
    const overIndex = playerIndexFromDealId(String(over.id));
    const oldIndex = tapOrder.indexOf(activeIndex);
    const newIndex = tapOrder.indexOf(overIndex);

    if (oldIndex < 0 || newIndex < 0) {
      return;
    }

    setTapOrder((currentOrder) => arrayMove(currentOrder, oldIndex, newIndex));
  }

  function handleChooseDealing() {
    if (dealMode === "tap") {
      onChooseDealing({
        dealType: "tap",
        cutSize: clampedCutSize,
        order: tapOrder,
      });
      return;
    }

    onChooseDealing({
      dealType: "round_robin",
      cutSize: clampedCutSize,
    });
  }

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
                <div className="grid gap-3 rounded-2xl border border-primary/30 bg-primary/5 p-4">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <p className="text-sm font-medium">Deal choice</p>
                    <Badge variant={isDealChooser ? "default" : "outline"}>
                      {isDealChooser ? "Your pick" : (dealChooserName ?? "Waiting")}
                    </Badge>
                  </div>
                  {isDealChooser ? (
                    <div className="grid gap-3">
                      <CutDeckControl
                        cutSize={cutSize}
                        clampedCutSize={clampedCutSize}
                        maxCutSize={maxCutSize}
                        onCutSizeChange={setCutSize}
                      />

                      <p className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
                        Dealing style
                      </p>
                      <div className="grid grid-cols-2 gap-2">
                        <Button
                          type="button"
                          variant={dealMode === "cut" ? "default" : "outline"}
                          onClick={() => setDealMode("cut")}
                        >
                          Round robin
                        </Button>
                        <Button
                          type="button"
                          variant={dealMode === "tap" ? "default" : "outline"}
                          onClick={() => setDealMode("tap")}
                        >
                          Tap order
                        </Button>
                      </div>

                      {dealMode === "tap" ? (
                        <div className="grid gap-2">
                          <p className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
                            Dealing order
                          </p>
                          <DndContext
                            collisionDetection={closestCenter}
                            onDragEnd={handleTapOrderDragEnd}
                          >
                            <SortableContext
                              items={tapOrderIds}
                              strategy={verticalListSortingStrategy}
                            >
                              <div className="grid gap-2">
                                {tapOrder.map((playerIndex, index) => {
                                  const player = players[playerIndex];

                                  if (!player) {
                                    return null;
                                  }

                                  return (
                                    <SortableDealOrderPlayer
                                      key={player.playerId}
                                      player={player}
                                      playerIndex={playerIndex}
                                      position={index}
                                    />
                                  );
                                })}
                              </div>
                            </SortableContext>
                          </DndContext>
                        </div>
                      ) : null}

                      <Button type="button" onClick={handleChooseDealing}>
                        Start round
                      </Button>
                    </div>
                  ) : (
                    <div className="grid gap-1 text-sm text-muted-foreground">
                      <p>
                        Waiting for{" "}
                        <span className="font-medium text-foreground">
                          {dealChooserName ?? "the deal chooser"}
                        </span>{" "}
                        to cut the deck and choose the dealing style.
                      </p>
                      {dealerName ? <p>Dealer: {dealerName}</p> : null}
                    </div>
                  )}
                </div>
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
