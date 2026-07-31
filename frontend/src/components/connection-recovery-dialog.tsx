import { type LobbyState, useGameWebSocket } from "#/components/game-websocket-provider";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "#/components/ui/dialog";
import { Spinner } from "#/components/ui/spinner";
import { Caption } from "#/components/typography";
import { m } from "#/paraglide/messages.js";

type ConnectionStatus = LobbyState["connectionStatus"];

export function ConnectionRecoveryDialog({
  connectionStatus,
}: {
  connectionStatus: ConnectionStatus;
}) {
  return (
    <Dialog open={connectionStatus === "disconnected"}>
      <DialogContent
        showCloseButton={false}
        className="max-w-xs justify-items-center gap-5 p-5 text-center sm:max-w-sm sm:p-6"
      >
        <div className="relative flex size-14 items-center justify-center" aria-hidden="true">
          <span className="bg-primary/10 ring-primary/20 absolute inset-0 rounded-full ring-1" />
          <span className="bg-primary/10 flex size-11 items-center justify-center rounded-full">
            <Spinner className="text-primary size-5" />
          </span>
        </div>
        <DialogHeader className="items-center gap-2.5">
          <DialogTitle>{m.reconnecting_game()}</DialogTitle>
          <DialogDescription className="max-w-xs text-balance">
            {m.connection_interrupted_description()}
          </DialogDescription>
          <Caption className="max-w-xs text-balance" role="note">
            {m.connection_long_wait_hint()}
          </Caption>
        </DialogHeader>
      </DialogContent>
    </Dialog>
  );
}

export function GlobalConnectionRecoveryDialog() {
  const { state } = useGameWebSocket();

  return <ConnectionRecoveryDialog connectionStatus={state.connectionStatus} />;
}
