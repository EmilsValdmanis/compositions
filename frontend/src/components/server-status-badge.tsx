import { useQuery } from "@tanstack/react-query";
import { Badge } from "./ui/badge";
import { Spinner } from "./ui/spinner";
import { checkGameServerHealth } from "#/lib/health";

const pollIntervalMs = 15000;

const variantMap = {
  online: "default",
  offline: "destructive",
  checking: "outline",
} as const;

export function ServerStatusBadge() {
  const { data } = useQuery({
    queryKey: ["game-server-health"],
    queryFn: checkGameServerHealth,
    refetchInterval: pollIntervalMs,
    refetchOnWindowFocus: true,
    refetchOnReconnect: true,
  });

  const status = data?.status ?? "checking";
  const variant = variantMap[status];

  return (
    <Badge variant={variant} className="capitalize">
      {status === "checking" && <Spinner className="inline-start" />}
      {status}
    </Badge>
  );
}
