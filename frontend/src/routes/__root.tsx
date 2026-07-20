import { HeadContent, createRootRouteWithContext } from "@tanstack/react-router";
import { createServerFn } from "@tanstack/react-start";
import { getCookie, getRequestHeaders, getRequestUrl } from "@tanstack/react-start/server";
import type { QueryClient } from "@tanstack/react-query";
import { z } from "zod";
import {
  GlobalErrorComponent,
  GlobalNotFoundComponent,
} from "#/components/routes/global-route-status";
import { RootDocument } from "#/components/routes/root-document";
import { auth } from "#/lib/auth";
import { createSocialMeta, defaultSocialDescription } from "#/lib/social-meta";
import { loadVerifiedSession } from "#/lib/verified-session";
import appCss from "../styles.css?url";
import { m } from "#/paraglide/messages.js";

const rootHeadContentMarker = <HeadContent />;
void rootHeadContentMarker;

const getAppContext = createServerFn({ method: "GET" })
  .validator(z.undefined())
  .handler(async () => {
    const headers = new Headers(getRequestHeaders());

    return {
      session: await loadVerifiedSession(headers, auth.api.getSession),
      sidebarOpen: getCookie("sidebar_state"),
      siteOrigin: getRequestUrl({ xForwardedHost: true }).origin,
    };
  });

export const Route = createRootRouteWithContext<{
  queryClient: QueryClient;
}>()({
  beforeLoad: async () => {
    return getAppContext();
  },
  head: ({ match }) => {
    return {
      meta: [
        {
          charSet: "utf-8",
        },
        {
          name: "viewport",
          content: "width=device-width, initial-scale=1",
        },
        {
          title: m.page_title(),
        },
        { name: "application-name", content: m.app_name() },
        { name: "theme-color", content: "#0069a8" },
        ...createSocialMeta({
          title: m.page_title(),
          description: defaultSocialDescription(),
          origin: match.context.siteOrigin,
        }),
      ],
      links: [
        {
          rel: "stylesheet",
          href: appCss,
        },
        {
          rel: "icon",
          type: "image/svg+xml",
          href: "/favicon.svg",
        },
        {
          rel: "icon",
          href: "/favicon.ico",
          sizes: "any",
        },
        {
          rel: "icon",
          type: "image/png",
          sizes: "96x96",
          href: "/favicon-96x96.png",
        },
        {
          rel: "icon",
          type: "image/png",
          sizes: "48x48",
          href: "/favicon-48x48.png",
        },
        {
          rel: "icon",
          type: "image/png",
          sizes: "32x32",
          href: "/favicon-32x32.png",
        },
        {
          rel: "icon",
          type: "image/png",
          sizes: "16x16",
          href: "/favicon-16x16.png",
        },
        {
          rel: "apple-touch-icon",
          sizes: "180x180",
          href: "/apple-touch-icon.png",
        },
        {
          rel: "manifest",
          href: "/site.webmanifest",
        },
      ],
    };
  },
  errorComponent: GlobalErrorComponent,
  notFoundComponent: GlobalNotFoundComponent,
  shellComponent: RootDocument,
});
