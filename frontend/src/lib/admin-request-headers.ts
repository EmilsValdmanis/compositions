import { getRequestHeaders } from "@tanstack/react-start/server";

export function adminRequestHeaders() {
  const requestHeaders = new Headers(getRequestHeaders());
  const headers = new Headers({ accept: "application/json" });
  const cookie = requestHeaders.get("cookie");
  if (cookie) headers.set("cookie", cookie);
  return headers;
}
