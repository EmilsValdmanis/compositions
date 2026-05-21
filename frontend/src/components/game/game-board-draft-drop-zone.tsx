import { type ReactNode } from "react";
import { useDroppable } from "@dnd-kit/core";
import { cn } from "#/lib/utils";

export function GameBoardDraftDropZone({
  id,
  className,
  children,
}: {
  id: string;
  className?: string;
  children: ReactNode;
}) {
  const { setNodeRef, isOver } = useDroppable({ id });

  return (
    <div
      ref={setNodeRef}
      className={cn(
        className,
        isOver ? "border-primary bg-primary/10 ring-1 ring-primary/30" : undefined,
      )}
    >
      {children}
    </div>
  );
}
