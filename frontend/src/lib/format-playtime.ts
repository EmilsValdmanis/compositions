import { m } from "#/paraglide/messages.js";

export function formatPlaytime(totalSeconds: number) {
  const totalMinutes = Math.floor(totalSeconds / 60);
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  return hours > 0
    ? m.playtime_hours_minutes({ hours, minutes })
    : m.playtime_minutes({ minutes: totalMinutes });
}
