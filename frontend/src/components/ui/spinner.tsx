import { cn } from "@/lib/utils";
import { HugeiconsIcon } from "@hugeicons/react";
import { Loading03Icon } from "@hugeicons/core-free-icons";
import { m } from "#/paraglide/messages.js";

function Spinner({ className, strokeWidth, ...props }: React.ComponentProps<"svg">) {
  return (
    <HugeiconsIcon
      icon={Loading03Icon}
      strokeWidth={strokeWidth !== undefined ? Number(strokeWidth) : 2}
      data-slot="spinner"
      role="status"
      aria-label={m.loading()}
      className={cn("size-4 animate-spin", className)}
      {...props}
    />
  );
}

export { Spinner };
