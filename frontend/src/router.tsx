import { createRouter as createTanStackRouter } from "@tanstack/react-router";
import { routeTree } from "./routeTree.gen";
import { QueryClient } from "@tanstack/react-query";
import { setupRouterSsrQueryIntegration } from "@tanstack/react-router-ssr-query";
import * as Sentry from "@sentry/tanstackstart-react";

export function getRouter() {
  const queryClient = new QueryClient();

  const router = createTanStackRouter({
    routeTree,
    context: { queryClient },
    scrollRestoration: true,
    defaultPreload: "intent",
    defaultPreloadStaleTime: 0,
  });
  setupRouterSsrQueryIntegration({
    router,
    queryClient,
  });

  if (!router.isServer) {
    Sentry.init({
      dsn: "https://1b571087a2847838e64e3a7856ee9533@o4511438083653632.ingest.de.sentry.io/4511438085161040",
      enabled: process.env.NODE_ENV === "production",

      // Adds request headers and IP for users, for more info visit:
      // https://docs.sentry.io/platforms/javascript/guides/tanstackstart-react/configuration/options/#sendDefaultPii
      sendDefaultPii: true,

      integrations: [],
    });
  }

  return router;
}

declare module "@tanstack/react-router" {
  interface Register {
    router: ReturnType<typeof getRouter>;
  }
}
