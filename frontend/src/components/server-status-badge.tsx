import { useQuery } from "@tanstack/react-query";
import { BadgeInfoIcon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { checkGameServerHealth } from "#/lib/health";
import { H6, Text } from "#/components/typography";
import { useGameWebSocket } from "./game-websocket-provider";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { Label } from "./ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "./ui/popover";
import { Separator } from "./ui/separator";
import { Spinner } from "./ui/spinner";

const connectionLabels = {
  idle: "Checking",
  connecting: "Connecting",
  connected: "Connected",
  disconnected: "Disconnected",
} as const;

const serverVariants = {
  checking: "outline",
  online: "default",
  offline: "destructive",
} as const;

export function ServerStatusBadge() {
  const { state, connect, disconnect } = useGameWebSocket();

  const { data, isFetching, isPending } = useQuery({
    queryKey: ["game-server-health"],
    queryFn: checkGameServerHealth,
    refetchOnWindowFocus: true,
    refetchOnReconnect: true,
  });

  const serverStatus = data?.status ?? "checking";
  const isCheckingServer = isFetching || isPending;

  const hasConnectionIssue = state.connectionStatus === "disconnected";
  const hasServerIssue = !isCheckingServer && serverStatus === "offline";
  const hasIssue = hasConnectionIssue || hasServerIssue;

  const connectionVariant =
    state.connectionStatus === "connected"
      ? "default"
      : hasConnectionIssue
        ? "destructive"
        : "secondary";

  return (
    <Popover>
      <PopoverTrigger render={<Button variant="outline" size="icon" className="relative" />}>
        <HugeiconsIcon icon={BadgeInfoIcon} />
        {hasIssue && (
          <span className="absolute right-0 top-0 flex size-2.5">
            <span className="bg-destructive absolute inline-flex h-full w-full animate-ping rounded-full opacity-75" />
            <span className="bg-destructive relative inline-flex size-2.5 rounded-full" />
          </span>
        )}
      </PopoverTrigger>
      <PopoverContent align="start">
        <div className="space-y-1">
          <H6>Connection Status</H6>
          <Text as="p" variant="caption">
            Real-time server and connection diagnostics.
          </Text>
        </div>
        <Separator />
        <div className="space-y-2">
          <div className="flex items-center justify-between gap-4">
            <Text as={Label} variant="caption">
              Connection
            </Text>
            <Badge variant={connectionVariant}>{connectionLabels[state.connectionStatus]}</Badge>
          </div>
          <div className="flex items-center justify-between gap-4">
            <Text as={Label} variant="caption">
              Server
            </Text>
            <Badge variant={serverVariants[serverStatus]} className="gap-1">
              {isCheckingServer && <Spinner className="size-3" />}
              <Text variant="status">{serverStatus}</Text>
            </Badge>
          </div>
        </div>
        {state.connectionStatus === "disconnected" ? (
          <Button type="button" onClick={() => void connect()} disabled={isCheckingServer}>
            Reconnect
          </Button>
        ) : state.connectionStatus !== "idle" ? (
          <Button type="button" variant="outline" onClick={disconnect}>
            Disconnect
          </Button>
        ) : null}
      </PopoverContent>
    </Popover>
  );
}
