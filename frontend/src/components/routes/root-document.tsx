import { useEffect } from "react";
import { HeadContent, Scripts } from "@tanstack/react-router";
import { TanStackRouterDevtoolsPanel } from "@tanstack/react-router-devtools";
import { ReactQueryDevtoolsPanel } from "@tanstack/react-query-devtools";
import { TanStackDevtools } from "@tanstack/react-devtools";
import { GameWebSocketProvider } from "#/components/game-websocket-provider";
import { ThemeProvider } from "#/components/theme-provider";
import { AutoConnectWebSocket } from "#/components/routes/auto-connect-websocket";
import { ThemeAwareToaster } from "#/components/routes/theme-aware-toaster";

export function RootDocument({ children }: { children: React.ReactNode }) {
  useEffect(() => {
    if (import.meta.env.PROD) {
      void import("@vercel/analytics").then(({ inject }) => inject());
      void import("@vercel/speed-insights").then(({ injectSpeedInsights }) =>
        injectSpeedInsights(),
      );
    }
  }, []);

  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <HeadContent />
        {import.meta.env.DEV && (
          <script crossOrigin="anonymous" defer src="//unpkg.com/react-scan/dist/auto.global.js" />
        )}
      </head>
      <body>
        <div className="flex h-dvh w-full flex-col">
          <ThemeProvider defaultTheme="system" storageKey="theme">
            <GameWebSocketProvider>
              {children}
              <AutoConnectWebSocket />
            </GameWebSocketProvider>
            <ThemeAwareToaster />
          </ThemeProvider>
          <TanStackDevtools
            config={{
              position: "bottom-right",
            }}
            plugins={[
              {
                name: "Tanstack Router",
                render: <TanStackRouterDevtoolsPanel />,
              },
              {
                name: "Tanstack Query",
                render: <ReactQueryDevtoolsPanel />,
              },
            ]}
          />
          <Scripts />
        </div>
      </body>
    </html>
  );
}
