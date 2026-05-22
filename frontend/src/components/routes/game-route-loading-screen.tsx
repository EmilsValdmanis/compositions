import { Spinner } from "#/components/ui/spinner";

export function GameRouteLoadingScreen() {
  return (
    <section className="flex flex-1 items-center justify-center">
      <div className="flex flex-col items-center gap-3 text-center">
        <Spinner className="size-8" />
        <div className="space-y-1">
          <p className="text-sm font-medium">Reconnecting to your game</p>
          <p className="text-muted-foreground text-sm">Loading the latest room state…</p>
        </div>
      </div>
    </section>
  );
}
