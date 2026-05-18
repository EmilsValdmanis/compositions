import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "#/components/ui/card";

export function DebugJsonView({ value }: { value: unknown }) {
  return (
    <Card className="h-full max-h-[calc(100vh-10rem)] shadow-sm">
      <CardHeader>
        <CardTitle>Debug</CardTitle>
        <CardDescription>Live websocket state</CardDescription>
      </CardHeader>
      <CardContent className="min-h-0 grow pb-6">
        <pre className="h-full min-h-96 overflow-auto rounded-2xl border border-border bg-muted/30 p-3 text-xs leading-relaxed text-muted-foreground">
          {JSON.stringify(value, null, 2)}
        </pre>
      </CardContent>
    </Card>
  );
}
