import { type ReactNode } from "react";
import { useDroppable } from "@dnd-kit/core";
import { cn } from "#/lib/utils";

export function GameBoardDraftDropZone({
  id,
  className,
  activeClassName = "border-primary bg-primary/10 ring-1 ring-primary/30",
  children,
  invalid,
}: {
  id: string;
  className?: string;
  activeClassName?: string | null;
  children: ReactNode;
  invalid?: boolean;
}) {
  const { setNodeRef, isOver } = useDroppable({ id });

  return (
    <div
      ref={setNodeRef}
      data-over={isOver ? "true" : "false"}
      aria-invalid={invalid || undefined}
      className={cn(className, isOver ? activeClassName : undefined)}
    >
      {children}
    </div>
  );
}
