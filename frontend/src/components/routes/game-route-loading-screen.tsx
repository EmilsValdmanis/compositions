import { Spinner } from "#/components/ui/spinner";
import { H6, P } from "#/components/typography";

export function GameRouteLoadingScreen() {
  return (
    <section className="flex flex-1 items-center justify-center">
      <div className="flex flex-col items-center gap-3 text-center">
        <Spinner className="size-8" />
        <div className="space-y-1">
          <H6>Reconnecting to your game</H6>
          <P size="sm" className="text-muted-foreground">
            Loading the latest room state…
          </P>
        </div>
      </div>
    </section>
  );
}
