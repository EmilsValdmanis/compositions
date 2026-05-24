import { HeadContent, createRootRouteWithContext } from "@tanstack/react-router";
import { getRequestHeaders } from "@tanstack/react-start/server";
import type { QueryClient } from "@tanstack/react-query";
import { createServerFn } from "@tanstack/react-start";
import { auth } from "#/lib/auth";
import { loadVerifiedSession } from "#/lib/verified-session";
import {
  GlobalErrorComponent,
  GlobalNotFoundComponent,
} from "#/components/routes/global-route-status";
import { RootDocument } from "#/components/routes/root-document";
import { z } from "zod";
import appCss from "../styles.css?url";

const rootHeadContentMarker = <HeadContent />;
void rootHeadContentMarker;

const getSession = createServerFn({ method: "GET" })
  .inputValidator(z.undefined())
  .handler(async () => {
    const headers = new Headers(getRequestHeaders());
    return loadVerifiedSession(headers, auth.api.getSession);
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
  errorComponent: GlobalErrorComponent,
  notFoundComponent: GlobalNotFoundComponent,
  shellComponent: RootDocument,
});
