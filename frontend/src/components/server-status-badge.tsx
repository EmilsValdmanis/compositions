import { useQuery } from "@tanstack/react-query";
import { HugeiconsIcon } from "@hugeicons/react";
import { BadgeInfoIcon } from "@hugeicons/core-free-icons";
import { checkGameServerHealth } from "#/lib/health";
import { useGameWebSocket } from "./game-websocket-provider";
import { Button } from "./ui/button";
import { Badge } from "./ui/badge";
import { Spinner } from "./ui/spinner";
import { Separator } from "./ui/separator";
import { Label } from "./ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "./ui/popover";
import { cn } from "#/lib/utils";

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

export function ServerStatusBadge({ className }: { className?: string }) {
  const { state } = useGameWebSocket();

  const { data, isFetching, isPending } = useQuery({
    queryKey: ["game-server-health"],
    queryFn: checkGameServerHealth,
    enabled: typeof window !== "undefined",
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
      <PopoverTrigger asChild>
        <Button variant="outline" size="icon" className={cn("relative", className)}>
          <HugeiconsIcon icon={BadgeInfoIcon} />
          {hasIssue && (
            <span className="absolute right-0 top-0 flex size-2.5">
              <span className="bg-destructive absolute inline-flex h-full w-full animate-ping rounded-full opacity-75" />
              <span className="bg-destructive relative inline-flex size-2.5 rounded-full" />
            </span>
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start">
        <div className="space-y-1">
          <h4 className="text-sm font-semibold">Connection Status</h4>
          <p className="text-muted-foreground text-xs">
            Real-time server and connection diagnostics.
          </p>
        </div>
        <Separator />
        <div className="space-y-2">
          <div className="flex items-center justify-between gap-4">
            <Label className="text-muted-foreground text-xs">Connection</Label>
            <Badge variant={connectionVariant}>{connectionLabels[state.connectionStatus]}</Badge>
          </div>
          <div className="flex items-center justify-between gap-4">
            <Label className="text-muted-foreground text-xs">Server</Label>
            <Badge variant={serverVariants[serverStatus]} className="capitalize gap-1">
              {isCheckingServer && <Spinner className="size-3" />}
              {serverStatus}
            </Badge>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
