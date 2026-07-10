import { GameRulesGuide } from "#/components/game/game-rules-guide";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "#/components/ui/dialog";

type GameRulesDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function GameRulesDialog({ open, onOpenChange }: GameRulesDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="w-[min(58rem,calc(100vw-2rem))] max-w-none gap-0 overflow-hidden p-0 sm:max-w-none">
        <DialogHeader className="border-b border-border/70 px-6 py-5 pr-14">
          <DialogTitle>Rules</DialogTitle>
          <DialogDescription>
            A quick guide to turns, compositions, jokers, scoring, and winning.
          </DialogDescription>
        </DialogHeader>
        <div className="max-h-[min(72dvh,48rem)] overflow-y-auto px-6 py-5">
          <GameRulesGuide />
        </div>
      </DialogContent>
    </Dialog>
  );
}
