import { createServerFn } from "@tanstack/react-start";
import { setResponseHeader } from "@tanstack/react-start/server";
import { readRequestSession } from "#/lib/auth";

export const hasGameSession = createServerFn({ method: "POST" }).handler(async () => {
  setResponseHeader("cache-control", "no-store");

  return (await readRequestSession()) !== null;
});
