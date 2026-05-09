import { useState } from "react";
import { createFileRoute } from "@tanstack/react-router";
import { useGameWebSocket } from "#/components/game-websocket-provider";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "#/components/ui/card";
import { Input } from "#/components/ui/input";
import { Label } from "#/components/ui/label";
import { ServerStatusBadge } from "#/components/server-status-badge";

export const Route = createFileRoute("/")({ component: Home });

function Home() {
  const { state, isConnected, connect, disconnect, createRoom, joinRoom, startGame, leaveRoom } =
    useGameWebSocket();
  const [createName, setCreateName] = useState("");
  const [joinName, setJoinName] = useState("");
  const [joinRoomCode, setJoinRoomCode] = useState("");

  return (
    <main className="min-h-screen bg-background px-4 py-8 md:px-8">
      <div className="mx-auto flex w-full max-w-6xl flex-col gap-6">
        <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div className="flex items-center gap-3">
            <h1 className="text-2xl">Compositions</h1>
          </div>
          <div className="flex items-center gap-2">
            <ServerStatusBadge />
          </div>
        </div>

        <div className="grid gap-6 lg:grid-cols-[1fr_1fr]">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle>Connection</CardTitle>
                <Badge variant={isConnected ? "default" : "outline"}>
                  {state.connectionStatus}
                </Badge>
              </div>
              <CardDescription>Connect first, then create or join a room.</CardDescription>
            </CardHeader>
            <CardContent className="grid gap-4">
              <div className="grid gap-2">
                <Label htmlFor="session-id">Session ID</Label>
                <Input id="session-id" value={state.sessionId} readOnly disabled={!isConnected} />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="player-id">Player ID</Label>
                <Input id="player-id" value={state.playerId} readOnly disabled={!isConnected} />
              </div>
              <div className="flex flex-wrap gap-2">
                <Button
                  onClick={connect}
                  disabled={state.connectionStatus === "connecting" || isConnected}
                >
                  Connect
                </Button>
                <Button
                  variant="outline"
                  onClick={disconnect}
                  disabled={state.connectionStatus === "disconnected"}
                >
                  Disconnect
                </Button>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Room</CardTitle>
              <CardDescription>Minimal lobby controls only.</CardDescription>
            </CardHeader>
            <CardContent className="flex flex-col gap-6">
              <div className="grid gap-3">
                <Label htmlFor="create-name">Create room name</Label>
                <Input
                  id="create-name"
                  value={createName}
                  onChange={(event) => setCreateName(event.target.value)}
                  placeholder="Host"
                />
                <Button onClick={() => createRoom(createName)} disabled={!isConnected}>
                  Create Room
                </Button>
              </div>

              <div className="grid gap-3">
                <Label htmlFor="join-room-code">Join room code</Label>
                <Input
                  id="join-room-code"
                  value={joinRoomCode}
                  onChange={(event) => setJoinRoomCode(event.target.value.toUpperCase())}
                  placeholder="ABC123"
                />
                <Label htmlFor="join-name">Join room name</Label>
                <Input
                  id="join-name"
                  value={joinName}
                  onChange={(event) => setJoinName(event.target.value)}
                  placeholder="Guest"
                />
                <Button
                  variant="secondary"
                  onClick={() => joinRoom(joinRoomCode, joinName)}
                  disabled={!isConnected}
                >
                  Join Room
                </Button>
              </div>

              <div className="flex flex-wrap gap-2">
                <Button onClick={startGame} disabled={!isConnected || !state.room}>
                  Start Game
                </Button>
                <Button
                  variant="destructive"
                  onClick={leaveRoom}
                  disabled={!isConnected || !state.room}
                >
                  Leave
                </Button>
              </div>

              <div className="rounded-3xl border bg-muted/40 p-4">
                <div className="flex flex-wrap items-center gap-2 text-sm">
                  <span>Room ID:</span>
                  <Badge variant="outline">{state.room?.code || "-"}</Badge>
                  <span>Phase:</span>
                  <Badge variant="secondary">{state.room?.phase || "none"}</Badge>
                </div>
                <div className="mt-4 grid gap-2">
                  {(state.room?.players ?? []).map((player) => (
                    <div
                      key={player.playerId}
                      className="flex flex-col gap-2 rounded-3xl border bg-muted px-3 py-2 text-sm"
                    >
                      <div className="flex items-center gap-2">
                        <Badge variant={player.connected ? "default" : "destructive"}>
                          seat {player.seat}
                        </Badge>
                        <span>{player.name}</span>
                        {player.isHost ? <Badge variant="outline">host</Badge> : null}
                      </div>
                      <span className="text-muted-foreground">{player.playerId}</span>
                    </div>
                  ))}
                  {state.room?.players.length ? null : (
                    <div className="text-sm text-muted-foreground">No room joined yet.</div>
                  )}
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>State</CardTitle>
            <CardDescription>Raw JSON view for debugging.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="mb-3 flex flex-wrap gap-2 text-sm">
              <Badge variant={state.lastError ? "destructive" : "outline"}>
                {state.lastError ?? "no error"}
              </Badge>
              <Badge variant="outline">last event: {state.lastEvent ?? "none"}</Badge>
            </div>
            <pre className="overflow-x-auto rounded-3xl border bg-muted/40 p-4 text-xs leading-6">
              {JSON.stringify(state, null, 2)}
            </pre>
          </CardContent>
        </Card>
      </div>
    </main>
  );
}
