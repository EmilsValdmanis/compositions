import { useState } from "react";
import {
  Agreement01Icon,
  Bug01Icon,
  Logout02Icon,
  MoreVerticalIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { toast } from "sonner";
import { useGameWebSocket } from "#/components/game-websocket-provider";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "#/components/ui/alert-dialog";
import { Button } from "#/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "#/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "#/components/ui/dropdown-menu";
import { Field, FieldDescription, FieldGroup, FieldLabel } from "#/components/ui/field";
import { Textarea } from "#/components/ui/textarea";

type OpenDialog = "forfeit" | "end" | "report" | null;

export function GameControlsMenu() {
  const { state, forfeitGame, requestEndGame, reportIssue } = useGameWebSocket();
  const [openDialog, setOpenDialog] = useState<OpenDialog>(null);
  const [description, setDescription] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const room = state.room;
  const isActiveGame = room?.phase === "in_progress" || room?.phase === "round_over";
  const currentPlayer = room?.players.find((player) => player.playerId === state.playerId);
  const activePlayers = room?.players.filter((player) => !player.forfeited) ?? [];
  const otherActivePlayers = activePlayers.filter((player) => player.playerId !== state.playerId);
  const hasActiveProposal = Boolean(room?.endProposal);

  if (!isActiveGame || !currentPlayer || currentPlayer.forfeited) {
    return null;
  }

  const forfeitConsequence =
    otherActivePlayers.length === 1
      ? `${otherActivePlayers[0]?.name ?? "The other player"} will win immediately.`
      : `The remaining ${otherActivePlayers.length} players will continue without you.`;

  async function runAction(action: () => Promise<unknown>, successMessage: string) {
    setIsSubmitting(true);
    try {
      await action();
      setOpenDialog(null);
      toast.success(successMessage);
    } catch {
      // The shared WebSocket error handler presents the server's reason.
    } finally {
      setIsSubmitting(false);
    }
  }

  async function submitReport(requestAbort: boolean) {
    const cleanDescription = description.trim();
    if (!cleanDescription) {
      toast.error("Describe what went wrong");
      return;
    }
    await runAction(
      () => reportIssue(cleanDescription, requestAbort),
      requestAbort ? "Report sent and abort requested" : "Problem report sent",
    );
    setDescription("");
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={<Button type="button" variant="ghost" size="icon" aria-label="Game options" />}
        >
          <HugeiconsIcon icon={MoreVerticalIcon} />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-56">
          <DropdownMenuGroup>
            <DropdownMenuItem disabled={hasActiveProposal} onClick={() => setOpenDialog("end")}>
              <HugeiconsIcon icon={Agreement01Icon} />
              Ask to end game
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setOpenDialog("report")}>
              <HugeiconsIcon icon={Bug01Icon} />
              Report a problem
            </DropdownMenuItem>
          </DropdownMenuGroup>
          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <DropdownMenuItem variant="destructive" onClick={() => setOpenDialog("forfeit")}>
              <HugeiconsIcon icon={Logout02Icon} />
              Forfeit and leave
            </DropdownMenuItem>
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>

      <AlertDialog
        open={openDialog === "forfeit"}
        onOpenChange={(open) => setOpenDialog(open ? "forfeit" : null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia>
              <HugeiconsIcon icon={Logout02Icon} />
            </AlertDialogMedia>
            <AlertDialogTitle>Forfeit this game?</AlertDialogTitle>
            <AlertDialogDescription>
              You cannot rejoin. Your remaining cards will be shuffled back into the draw pile.{" "}
              {forfeitConsequence}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isSubmitting}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={isSubmitting}
              onClick={() => void runAction(forfeitGame, "You forfeited the game")}
            >
              Forfeit and leave
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={openDialog === "end"}
        onOpenChange={(open) => setOpenDialog(open ? "end" : null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia>
              <HugeiconsIcon icon={Agreement01Icon} />
            </AlertDialogMedia>
            <AlertDialogTitle>Ask everyone to end the game?</AlertDialogTitle>
            <AlertDialogDescription>
              All active players must agree. If accepted, the game ends without a winner and no game
              result is recorded.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isSubmitting}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              disabled={isSubmitting}
              onClick={() => void runAction(requestEndGame, "End-game request sent")}
            >
              Send request
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Dialog
        open={openDialog === "report"}
        onOpenChange={(open) => setOpenDialog(open ? "report" : null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Report a game problem</DialogTitle>
            <DialogDescription>
              Describe the broken state. The current round, turn, and game snapshot are attached
              automatically.
            </DialogDescription>
          </DialogHeader>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="game-problem-description">What went wrong?</FieldLabel>
              <Textarea
                id="game-problem-description"
                value={description}
                onChange={(event) => setDescription(event.target.value)}
                placeholder="For example: the turn stayed stuck after drawing from the discard pile."
                maxLength={500}
                rows={5}
                disabled={isSubmitting}
              />
              <FieldDescription>{description.length}/500 characters</FieldDescription>
            </Field>
          </FieldGroup>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              disabled={isSubmitting}
              onClick={() => void submitReport(false)}
            >
              Send report
            </Button>
            <Button
              type="button"
              disabled={isSubmitting || hasActiveProposal}
              onClick={() => void submitReport(true)}
            >
              Report and request abort
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
