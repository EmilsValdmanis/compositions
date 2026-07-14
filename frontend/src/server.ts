import * as Sentry from "@sentry/tanstackstart-react";
import handler, { createServerEntry } from "@tanstack/react-start/server-entry";
import { paraglideMiddleware } from "./paraglide/server.js";

export default createServerEntry(
  Sentry.wrapFetchWithSentry({
    async fetch(request: Request): Promise<Response> {
      try {
        return await paraglideMiddleware(request, () => handler.fetch(request));
      } catch (error) {
        Sentry.captureException(error);
        throw error;
      }
    },
  }),
);
