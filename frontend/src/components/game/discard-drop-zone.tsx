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
      className={`rounded-3xl border border-dashed p-2.5 transition ${
        isOver && !disabled ? "border-primary bg-primary/10" : "border-border bg-muted/20"
      }`}
    >
      {children}
    </div>
  );
}
