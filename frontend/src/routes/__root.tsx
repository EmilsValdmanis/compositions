import { GameWebSocketProvider, useGameWebSocket } from "#/components/game-websocket-provider";
import { HeadContent, Scripts, createRootRouteWithContext } from "@tanstack/react-router";
import { useEffect } from "react";
import { TanStackRouterDevtoolsPanel } from "@tanstack/react-router-devtools";
import { ReactQueryDevtoolsPanel } from "@tanstack/react-query-devtools";
import { TanStackDevtools } from "@tanstack/react-devtools";
import { inject as InjectVercelAnalytics } from "@vercel/analytics";
import { injectSpeedInsights as InjectVercelSpeedInsights } from "@vercel/speed-insights";
import { ThemeProvider, useTheme } from "#/components/theme-provider";
import { getRequestHeaders } from "@tanstack/react-start/server";
import type { QueryClient } from "@tanstack/react-query";
import { createServerFn } from "@tanstack/react-start";
import { authClient } from "#/lib/auth-client";
import { Toaster } from "sonner";
import appCss from "../styles.css?url";

const getSession = createServerFn({ method: "GET" }).handler(async () => {
  const headers = getRequestHeaders();
  const { data: session } = await authClient.getSession({
    fetchOptions: {
      headers,
    },
  });
  return session;
});

export const Route = createRootRouteWithContext<{
  queryClient: QueryClient;
}>()({
  beforeLoad: async () => {
    const session = await getSession();
    return {
      session,
    };
  },
  head: () => ({
    meta: [
      {
        charSet: "utf-8",
      },
      {
        name: "viewport",
        content: "width=device-width, initial-scale=1",
      },
      {
        title: "Compositions",
      },
    ],
    links: [
      {
        rel: "stylesheet",
        href: appCss,
      },
    ],
  }),
  shellComponent: RootDocument,
});

function AutoConnectWebSocket() {
  const { connect, disconnect } = useGameWebSocket();
  const { session } = Route.useRouteContext();
  const isAuthenticated = !!session;

  useEffect(() => {
    if (isAuthenticated) {
      void connect();
    } else {
      disconnect();
    }
  }, [isAuthenticated]);

  return null;
}

function RootDocument({ children }: { children: React.ReactNode }) {
  InjectVercelAnalytics();
  InjectVercelSpeedInsights();

  const { theme } = useTheme();

  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <HeadContent />
        {import.meta.env.DEV && (
          <script crossOrigin="anonymous" src="//unpkg.com/react-scan/dist/auto.global.js" />
        )}
      </head>
      <body>
        <main className="min-h-screen flex items-center gap-4 flex-col">
          <ThemeProvider defaultTheme="system" storageKey="theme">
            <GameWebSocketProvider>
              {children}
              <AutoConnectWebSocket />
            </GameWebSocketProvider>
            <Toaster richColors theme={theme} />
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
        </main>
      </body>
    </html>
  );
}
