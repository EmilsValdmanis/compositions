import { type GameSnapshot } from "#/components/game-websocket-provider";
import { CompositionRow } from "#/components/game/composition-row";
import { Badge } from "#/components/ui/badge";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "#/components/ui/card";

export function GameBoardTable({ game }: { game: GameSnapshot | null }) {
  const tablePoints = (game?.activeCompositions ?? []).reduce(
    (total, composition) => total + composition.points,
    0,
  );

  return (
    <Card className="min-h-0 xl:flex-1">
      <CardHeader>
        <CardTitle>Table</CardTitle>
        <CardDescription>
          {game?.activeCompositions.length ?? 0} compositions on the table
        </CardDescription>
        <CardAction>
          <Badge variant="outline">{tablePoints} pts</Badge>
        </CardAction>
      </CardHeader>
      <CardContent className="grid h-full auto-rows-max gap-3">
        {game?.activeCompositions.length ? (
          game.activeCompositions.map((composition, index) => (
            <CompositionRow key={index} composition={composition} index={index} />
          ))
        ) : (
          <div className="grid min-h-56 place-items-center rounded-3xl border border-dashed border-border/70 text-sm text-muted-foreground">
            No compositions on the table.
          </div>
        )}
      </CardContent>
    </Card>
  );
}
