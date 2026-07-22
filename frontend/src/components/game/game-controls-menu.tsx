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
import { m } from "#/paraglide/messages.js";

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
      ? m.forfeit_other_wins({ name: otherActivePlayers[0]?.name ?? m.other_player() })
      : m.forfeit_players_continue({ count: otherActivePlayers.length });

  function runAction(action: () => Promise<unknown>, successMessage: string) {
    setIsSubmitting(true);
    return Promise.resolve()
      .then(action)
      .then(() => {
        setOpenDialog(null);
        toast.success(successMessage);
      })
      .catch(() => {
        // The shared WebSocket error handler presents the server's reason.
      })
      .finally(() => setIsSubmitting(false));
  }

  async function submitReport(requestAbort: boolean) {
    const cleanDescription = description.trim();
    if (!cleanDescription) {
      toast.error(m.describe_problem());
      return;
    }
    await runAction(
      () => reportIssue(cleanDescription, requestAbort),
      requestAbort ? m.report_abort_sent() : m.report_sent(),
    );
    setDescription("");
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button
              type="button"
              variant="ghost"
              size="icon"
              aria-label={m.game_controls()}
              title={m.game_controls()}
            />
          }
        >
          <HugeiconsIcon icon={MoreVerticalIcon} />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" side="bottom" className="w-56">
          <DropdownMenuGroup>
            <DropdownMenuItem disabled={hasActiveProposal} onClick={() => setOpenDialog("end")}>
              <HugeiconsIcon icon={Agreement01Icon} />
              {m.ask_end_game()}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setOpenDialog("report")}>
              <HugeiconsIcon icon={Bug01Icon} />
              {m.report_problem()}
            </DropdownMenuItem>
          </DropdownMenuGroup>
          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <DropdownMenuItem variant="destructive" onClick={() => setOpenDialog("forfeit")}>
              <HugeiconsIcon icon={Logout02Icon} />
              {m.forfeit_and_leave()}
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
            <AlertDialogTitle>{m.forfeit_title()}</AlertDialogTitle>
            <AlertDialogDescription>
              {m.forfeit_description({ consequence: forfeitConsequence })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isSubmitting}>{m.cancel()}</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={isSubmitting}
              onClick={() => void runAction(forfeitGame, m.forfeit_success())}
            >
              {m.forfeit_and_leave()}
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
            <AlertDialogTitle>{m.end_game_title()}</AlertDialogTitle>
            <AlertDialogDescription>{m.end_game_description()}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isSubmitting}>{m.cancel()}</AlertDialogCancel>
            <AlertDialogAction
              disabled={isSubmitting}
              onClick={() => void runAction(requestEndGame, m.end_request_sent())}
            >
              {m.send_request()}
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
            <DialogTitle>{m.report_title()}</DialogTitle>
            <DialogDescription>{m.report_description()}</DialogDescription>
          </DialogHeader>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="game-problem-description">{m.what_went_wrong()}</FieldLabel>
              <Textarea
                id="game-problem-description"
                value={description}
                onChange={(event) => setDescription(event.target.value)}
                placeholder={m.problem_placeholder()}
                maxLength={500}
                rows={5}
                disabled={isSubmitting}
              />
              <FieldDescription>
                {m.characters_count({ count: description.length })}
              </FieldDescription>
            </Field>
          </FieldGroup>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              disabled={isSubmitting}
              onClick={() => void submitReport(false)}
            >
              {m.send_report()}
            </Button>
            <Button
              type="button"
              disabled={isSubmitting || hasActiveProposal}
              onClick={() => void submitReport(true)}
            >
              {m.report_and_abort()}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
