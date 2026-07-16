import { m } from "#/paraglide/messages.js";

export function pageTitle(title: string) {
  return `${title} - ${m.app_name()}`;
}
