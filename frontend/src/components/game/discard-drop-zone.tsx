import { type ReactNode } from "react";
import { useDroppable } from "@dnd-kit/core";

export function DiscardDropZone({
  disabled,
  children,
}: {
  disabled: boolean;
  children: ReactNode;
}) {
  const { setNodeRef, isOver } = useDroppable({ id: "discard-pile", disabled });

  return (
    <div
      ref={setNodeRef}
      className={`h-full min-w-0 rounded-2xl transition ${
        isOver && !disabled ? "bg-primary/10 ring-2 ring-primary/70" : "ring-0"
      }`}
    >
      {children}
    </div>
  );
}
