import { DISCARD_DROP_ID } from "#/components/game/board-drop-targets";
import { type ReactNode } from "react";
import { useDroppable } from "@dnd-kit/core";
import { cn } from "#/lib/utils";

export function DiscardDropZone({
  disabled,
  children,
}: {
  disabled: boolean;
  children: ReactNode;
}) {
  const { setNodeRef, isOver } = useDroppable({ id: DISCARD_DROP_ID, disabled });

  return (
    <div
      ref={setNodeRef}
      className={cn(
        "h-full min-w-0 rounded-2xl transition-all",
        isOver && !disabled ? "bg-primary/10 ring-2 ring-primary/70" : "ring-0",
      )}
    >
      {children}
    </div>
  );
}
