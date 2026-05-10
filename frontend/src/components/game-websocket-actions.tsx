import { useEffect, useState } from "react";

import { useGameWebSocket } from "#/components/game-websocket-provider";
import { Button } from "#/components/ui/button";
import { Input } from "#/components/ui/input";

export function GameWebSocketActions() {
  const { state, connect, disconnect, createRoom, joinRoom, leaveRoom, startGame } =
    useGameWebSocket();
  const [roomCode, setRoomCode] = useState("");

  useEffect(() => {
    if (state.room?.code) {
      setRoomCode(state.room.code);
    }
  }, [state.room?.code]);

  return (
    <section className="w-full max-w-3xl rounded-4xl border border-border/60 bg-card/70 p-5 backdrop-blur-sm">
      <div className="flex flex-wrap gap-2">
        <Button type="button" variant="default" onClick={() => void connect()}>
          {state.sessionId ? "Connect / Reconnect" : "Connect"}
        </Button>
        <Button type="button" variant="outline" onClick={disconnect}>
          Disconnect
        </Button>
        <Button type="button" variant="secondary" onClick={createRoom}>
          Create Room
        </Button>
        <Button type="button" variant="secondary" onClick={leaveRoom}>
          Leave Room
        </Button>
        <Button type="button" variant="secondary" onClick={startGame}>
          Start Game
        </Button>
      </div>

      <div className="mt-3 flex flex-col gap-2 sm:flex-row">
        <Input
          value={roomCode}
          onChange={(event) => setRoomCode(event.target.value)}
          placeholder="Room code"
        />
        <Button type="button" variant="outline" onClick={() => joinRoom(roomCode)}>
          Join Room
        </Button>
      </div>

      <pre className="mt-4 overflow-x-auto rounded-3xl bg-muted/60 p-4 text-xs leading-5">
        {JSON.stringify(state, null, 2)}
      </pre>
    </section>
  );
}
