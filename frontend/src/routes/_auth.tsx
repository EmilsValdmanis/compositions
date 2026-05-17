import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/_auth")({
  beforeLoad: ({ context }) => {
    if (context.session) {
      throw redirect({ to: "/" });
    }
  },
  component: AuthLayout,
});

function AuthLayout() {
  return (
    <>
      <div
        className="absolute inset-0 -z-1 bg-size-[20px_20px]"
        style={{
          backgroundImage: "radial-gradient(var(--primary) 1px, transparent 1px)",
        }}
      />
      <div className="pointer-events-none absolute inset-0 flex items-center justify-center bg-background mask-[radial-gradient(ellipse_at_center,transparent_20%,black)]"></div>

      <div className="flex flex-col items-center gap-4 justify-center grow">
        <p className="relative bg-linear-to-b from-foreground to-muted-foreground bg-clip-text text-4xl font-bold text-transparent sm:text-7xl">
          Compositions
        </p>
        <Outlet />
      </div>
    </>
  );
}
