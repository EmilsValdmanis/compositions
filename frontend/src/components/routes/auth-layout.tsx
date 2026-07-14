import { Outlet } from "@tanstack/react-router";
import { Text } from "#/components/typography";

export function AuthLayout() {
  return (
    <>
      <div
        className="absolute inset-0 -z-1 bg-size-[20px_20px]"
        style={{
          backgroundImage: "radial-gradient(var(--primary) 1px, transparent 1px)",
        }}
      />
      <div className="pointer-events-none absolute inset-0 flex items-center justify-center bg-background mask-[radial-gradient(ellipse_at_center,transparent_20%,black)]"></div>

      <div className="flex grow flex-col items-center justify-center gap-4">
        <Text
          as="h1"
          variant="display"
          className="relative bg-linear-to-b from-foreground to-muted-foreground bg-clip-text text-transparent"
        >
          Compositions
        </Text>
        <Outlet />
      </div>
    </>
  );
}
