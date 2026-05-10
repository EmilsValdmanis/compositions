import { useEffect, useState } from "react";

import { useGameWebSocket } from "#/components/game-websocket-provider";
import { Avatar, AvatarFallback, AvatarImage } from "#/components/ui/avatar";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "#/components/ui/card";
import { Input } from "#/components/ui/input";
import { getUserInitials } from "#/lib/utils";

type GameWebSocketActionsProps = {
  currentUser?: {
    image?: string | null;
  };
};

function formatLabel(value: string) {
  return value
    .split("_")
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function compactId(value: string) {
  if (!value) {
    return "Not assigned";
  }

  if (value.length <= 12) {
    return value;
  }

  return `${value.slice(0, 6)}...${value.slice(-4)}`;
}

export function GameWebSocketActions({ currentUser }: GameWebSocketActionsProps) {
  const { state, connect, disconnect, createRoom, joinRoom, leaveRoom, startGame } =
    useGameWebSocket();
  const [roomCode, setRoomCode] = useState("");
  const players = state.room?.players ?? [];
  const currentPlayer = players.find((player) => player.playerId === state.playerId) ?? null;
  const otherPlayers = players.filter((player) => player.playerId !== state.playerId);
  const connectedPlayers = players.filter((player) => player.connected).length;
  const phaseLabel = formatLabel(state.room?.phase ?? "waiting_for_room");
  const isHost = currentPlayer?.isHost ?? false;
  const canStartGame = Boolean(state.room) && isHost;
  const canCreateRoom = state.connectionStatus === "connected" && !state.room;
  const canJoinRoom =
    state.connectionStatus === "connected" && !state.room && roomCode.trim() !== "";

  useEffect(() => {
    if (state.room?.code) {
      setRoomCode(state.room.code);
    }
  }, [state.room?.code]);

  return (
    <section className="w-full max-w-5xl">
      <Card>
        <CardHeader className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
          <div className="space-y-3">
            <div className="space-y-1">
              <CardTitle>Game room</CardTitle>
              <CardDescription>
                Live room, player, and phase details from the websocket state.
              </CardDescription>
            </div>

            <div className="flex flex-wrap gap-2">
              <Badge variant={state.connectionStatus === "connected" ? "default" : "outline"}>
                {formatLabel(state.connectionStatus)}
              </Badge>
              <Badge variant="secondary">Phase: {phaseLabel}</Badge>
              <Badge variant="outline">Room: {state.room?.code ?? "No room"}</Badge>
              <Badge variant="outline">
                Players: {connectedPlayers}/{players.length || 0} connected
              </Badge>
            </div>
          </div>

          <div className="flex w-full flex-col gap-3 xl:w-auto xl:min-w-md">
            <div className="flex flex-wrap gap-2">
              <Button
                type="button"
                variant="default"
                onClick={() => void connect()}
                disabled={state.connectionStatus === "connecting"}
              >
                {state.sessionId ? "Connect / Reconnect" : "Connect"}
              </Button>
              <Button
                type="button"
                variant="outline"
                onClick={disconnect}
                disabled={state.connectionStatus === "disconnected"}
              >
                Disconnect
              </Button>
            </div>

            <div className="flex flex-wrap gap-2">
              <Button
                type="button"
                variant="secondary"
                onClick={createRoom}
                disabled={!canCreateRoom}
              >
                Create Room
              </Button>
              <Button type="button" variant="secondary" onClick={leaveRoom} disabled={!state.room}>
                Leave Room
              </Button>
              <Button
                type="button"
                variant="secondary"
                onClick={startGame}
                disabled={!canStartGame}
              >
                Start Game
              </Button>
            </div>

            <div className="flex flex-col gap-2 sm:flex-row">
              <Input
                value={roomCode}
                onChange={(event) => setRoomCode(event.target.value.toUpperCase())}
                placeholder="Room code"
                maxLength={6}
              />
              <Button
                type="button"
                variant="outline"
                onClick={() => joinRoom(roomCode.trim().toUpperCase())}
                disabled={!canJoinRoom}
              >
                Join Room
              </Button>
            </div>
          </div>
        </CardHeader>

        <CardContent className="grid gap-4 py-5 lg:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
          <div className="grid gap-4">
            <Card size="sm" className="border border-border/60 bg-background/70 py-0 shadow-none">
              <CardHeader className="py-4">
                <CardTitle>Your status</CardTitle>
                <CardDescription>
                  Current session details and your seat in the active room.
                </CardDescription>
              </CardHeader>
              <CardContent className="grid gap-4 pb-4">
                <div className="flex items-center gap-3 rounded-3xl border border-border/60 bg-muted/30 px-4 py-3">
                  <Avatar size="lg">
                    {currentPlayer?.playerId === state.playerId && currentUser?.image ? (
                      <AvatarImage src={currentUser.image} alt={currentPlayer.name} />
                    ) : null}
                    <AvatarFallback>
                      {getUserInitials(currentPlayer?.name ?? "Guest Player")}
                    </AvatarFallback>
                  </Avatar>
                  <div className="min-w-0 space-y-1">
                    <p className="truncate font-medium">{currentPlayer?.name ?? "Not in a room"}</p>
                    <div className="flex flex-wrap gap-2">
                      <Badge variant={currentPlayer?.connected ? "default" : "outline"}>
                        {currentPlayer?.connected ? "Connected" : "Disconnected"}
                      </Badge>
                      {isHost ? <Badge variant="secondary">Host</Badge> : null}
                      {currentPlayer?.canReconnect ? (
                        <Badge variant="outline">Can reconnect</Badge>
                      ) : null}
                    </div>
                  </div>
                </div>

                <dl className="grid gap-3 text-sm sm:grid-cols-2">
                  <div className="rounded-3xl border border-border/60 bg-muted/20 px-4 py-3">
                    <dt className="text-xs uppercase tracking-wide text-muted-foreground">Seat</dt>
                    <dd className="mt-1 font-medium">
                      {currentPlayer ? `#${currentPlayer.seat + 1}` : "Waiting"}
                    </dd>
                  </div>
                  <div className="rounded-3xl border border-border/60 bg-muted/20 px-4 py-3">
                    <dt className="text-xs uppercase tracking-wide text-muted-foreground">
                      Player ID
                    </dt>
                    <dd className="mt-1 font-medium">{compactId(state.playerId)}</dd>
                  </div>
                  <div className="rounded-3xl border border-border/60 bg-muted/20 px-4 py-3 sm:col-span-2">
                    <dt className="text-xs uppercase tracking-wide text-muted-foreground">
                      Session ID
                    </dt>
                    <dd className="mt-1 font-medium">{compactId(state.sessionId)}</dd>
                  </div>
                </dl>
              </CardContent>
            </Card>

            <Card size="sm" className="border border-border/60 bg-background/70 py-0 shadow-none">
              <CardHeader className="py-4">
                <CardTitle>Room summary</CardTitle>
                <CardDescription>
                  Core lobby state surfaced from the websocket payload.
                </CardDescription>
              </CardHeader>
              <CardContent className="grid gap-3 pb-4 text-sm sm:grid-cols-2">
                <div className="rounded-3xl border border-border/60 bg-muted/20 px-4 py-3">
                  <p className="text-xs uppercase tracking-wide text-muted-foreground">Room code</p>
                  <p className="mt-1 font-medium">{state.room?.code ?? "Not joined"}</p>
                </div>
                <div className="rounded-3xl border border-border/60 bg-muted/20 px-4 py-3">
                  <p className="text-xs uppercase tracking-wide text-muted-foreground">
                    Game phase
                  </p>
                  <p className="mt-1 font-medium">{phaseLabel}</p>
                </div>
                <div className="rounded-3xl border border-border/60 bg-muted/20 px-4 py-3">
                  <p className="text-xs uppercase tracking-wide text-muted-foreground">Host</p>
                  <p className="mt-1 font-medium">
                    {players.find((player) => player.playerId === state.room?.hostPlayerId)?.name ??
                      "Waiting"}
                  </p>
                </div>
                <div className="rounded-3xl border border-border/60 bg-muted/20 px-4 py-3">
                  <p className="text-xs uppercase tracking-wide text-muted-foreground">Dealer</p>
                  <p className="mt-1 font-medium">
                    {typeof state.room?.dealerIndex === "number"
                      ? `Seat #${state.room.dealerIndex + 1}`
                      : "Not assigned"}
                  </p>
                </div>
                <div className="rounded-3xl border border-border/60 bg-muted/20 px-4 py-3 sm:col-span-2">
                  <p className="text-xs uppercase tracking-wide text-muted-foreground">
                    Last event
                  </p>
                  <p className="mt-1 font-medium">
                    {state.lastEvent ? formatLabel(state.lastEvent) : "None"}
                  </p>
                </div>
                {state.lastError ? (
                  <div className="rounded-3xl border border-destructive/30 bg-destructive/10 px-4 py-3 text-destructive sm:col-span-2">
                    <p className="text-xs uppercase tracking-wide">Last error</p>
                    <p className="mt-1 font-medium">{state.lastError}</p>
                  </div>
                ) : null}
              </CardContent>
            </Card>
          </div>

          <Card size="sm" className="border border-border/60 bg-background/70 py-0 shadow-none">
            <CardHeader className="py-4">
              <CardTitle>Players</CardTitle>
              <CardDescription>
                Everyone currently tracked in the room, including role and connection status.
              </CardDescription>
            </CardHeader>
            <CardContent className="grid gap-3 pb-4">
              {players.length === 0 ? (
                <div className="rounded-3xl border border-dashed border-border/70 px-4 py-6 text-sm text-muted-foreground">
                  Connect first, then create or join a room to see player details here.
                </div>
              ) : (
                [currentPlayer, ...otherPlayers].filter(Boolean).map((player) => {
                  if (!player) {
                    return null;
                  }

                  const isCurrentPlayer = player.playerId === state.playerId;
                  const isDealer = player.seat === state.room?.dealerIndex;

                  return (
                    <div
                      key={player.playerId}
                      className="flex items-center justify-between gap-3 rounded-3xl border border-border/60 bg-muted/20 px-4 py-3"
                    >
                      <div className="flex min-w-0 items-center gap-3">
                        <Avatar>
                          {isCurrentPlayer && currentUser?.image ? (
                            <AvatarImage src={currentUser.image} alt={player.name} />
                          ) : null}
                          <AvatarFallback>{getUserInitials(player.name)}</AvatarFallback>
                        </Avatar>
                        <div className="min-w-0">
                          <div className="flex flex-wrap items-center gap-2">
                            <p className="truncate font-medium">{player.name}</p>
                            {isCurrentPlayer ? <Badge variant="secondary">You</Badge> : null}
                          </div>
                          <p className="text-xs text-muted-foreground">Seat #{player.seat + 1}</p>
                        </div>
                      </div>

                      <div className="flex flex-wrap justify-end gap-2">
                        <Badge variant={player.connected ? "default" : "outline"}>
                          {player.connected ? "Connected" : "Offline"}
                        </Badge>
                        {player.isHost ? <Badge variant="secondary">Host</Badge> : null}
                        {isDealer ? <Badge variant="outline">Dealer</Badge> : null}
                        {player.canReconnect ? (
                          <Badge variant="outline">Reconnectable</Badge>
                        ) : null}
                      </div>
                    </div>
                  );
                })
              )}
            </CardContent>
          </Card>
        </CardContent>
      </Card>
    </section>
  );
}
