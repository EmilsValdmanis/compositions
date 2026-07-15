import { useEffect, useState } from "react";
import { Agreement01Icon, Bug01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { useGameWebSocket } from "#/components/game-websocket-provider";
import { Alert, AlertDescription, AlertTitle } from "#/components/ui/alert";
import { AnimatedNumber } from "#/components/ui/animated-number";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { P } from "#/components/typography";
import { m } from "#/paraglide/messages.js";

export function EndGameProposalAlert() {
  const { state, voteEndGame } = useGameWebSocket();
  const proposal = state.room?.endProposal;
  const [isVoting, setIsVoting] = useState(false);
  const [expiredProposalId, setExpiredProposalId] = useState<string | null>(null);

  useEffect(() => {
    if (!proposal) {
      return;
    }
    const delay = Math.max(0, new Date(proposal.expiresAt).getTime() - Date.now());
    const timeout = window.setTimeout(() => setExpiredProposalId(proposal.id), delay);
    return () => window.clearTimeout(timeout);
  }, [proposal]);

  if (!proposal || proposal.id === expiredProposalId) {
    return null;
  }

  const proposer = state.room?.players.find(
    (player) => player.playerId === proposal.proposerPlayerId,
  );
  const hasAgreed = proposal.agreedPlayerIds.includes(state.playerId);
  const isTechnicalAbort = proposal.kind === "technical_abort";
  const proposalId = proposal.id;

  function vote(approve: boolean) {
    setIsVoting(true);
    void Promise.resolve()
      .then(() => voteEndGame(proposalId, approve))
      .catch(() => {
        // The shared WebSocket error handler presents the server's reason.
      })
      .finally(() => setIsVoting(false));
  }

  return (
    <Alert>
      <HugeiconsIcon icon={isTechnicalAbort ? Bug01Icon : Agreement01Icon} />
      <AlertTitle>
        {isTechnicalAbort ? m.technical_abort_requested() : m.end_by_agreement()}
      </AlertTitle>
      <AlertDescription>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex min-w-0 flex-col gap-1">
            <P>
              {proposer?.name ?? m.a_player()}{" "}
              {isTechnicalAbort ? m.reported_game_problem() : m.would_end_without_winner()}.
            </P>
            {proposal.description ? <P className="truncate">“{proposal.description}”</P> : null}
          </div>
          <div className="flex shrink-0 items-center gap-2">
            <Badge variant="outline">
              <AnimatedNumber value={proposal.agreedPlayerIds.length} />/
              <AnimatedNumber value={proposal.eligiblePlayerIds.length} /> {m.agreed()}
            </Badge>
            {hasAgreed ? (
              <Badge variant="secondary">{m.waiting_for_others()}</Badge>
            ) : (
              <>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  disabled={isVoting}
                  onClick={() => vote(false)}
                >
                  {m.keep_playing()}
                </Button>
                <Button type="button" size="sm" disabled={isVoting} onClick={() => vote(true)}>
                  {isTechnicalAbort ? m.abort_game() : m.end_game()}
                </Button>
              </>
            )}
          </div>
        </div>
      </AlertDescription>
    </Alert>
  );
}
