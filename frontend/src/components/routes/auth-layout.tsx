import { Outlet } from "@tanstack/react-router";
import { H1, P } from "#/components/typography";
import { LanguageSwitcher } from "#/components/language-switcher";
import { m } from "#/paraglide/messages.js";

export function AuthLayout() {
  return (
    <>
      <LanguageSwitcher className="absolute top-4 right-4 z-10" />
      <div
        className="absolute inset-0 -z-1 bg-size-[20px_20px]"
        style={{
          backgroundImage: "radial-gradient(var(--primary) 1px, transparent 1px)",
        }}
      />
      <div className="pointer-events-none absolute inset-0 flex items-center justify-center bg-background mask-[radial-gradient(ellipse_at_center,transparent_20%,black)]"></div>

      <div className="flex grow flex-col items-center justify-center gap-6 px-6">
        <div className="flex max-w-2xl flex-col items-center gap-3 text-center">
          <H1 className="relative text-4xl/none font-bold sm:text-7xl/none bg-linear-to-b from-foreground to-muted-foreground bg-clip-text text-transparent">
            {m.app_name()}
          </H1>
          <P className="max-w-xl text-muted-foreground">{m.social_description()}</P>
        </div>
        <Outlet />
      </div>
    </>
  );
}
