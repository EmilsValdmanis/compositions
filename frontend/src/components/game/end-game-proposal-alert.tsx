import { useEffect, useState } from "react";
import { Agreement01Icon, Bug01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { useGameWebSocket } from "#/components/game-websocket-provider";
import { Alert, AlertDescription, AlertTitle } from "#/components/ui/alert";
import { Badge } from "#/components/ui/badge";
import { AnimatedNumber } from "#/components/ui/animated-number";
import { Button } from "#/components/ui/button";
import { P } from "#/components/typography";

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

  async function vote(approve: boolean) {
    setIsVoting(true);
    try {
      await voteEndGame(proposal!.id, approve);
    } catch {
      // The shared WebSocket error handler presents the server's reason.
    } finally {
      setIsVoting(false);
    }
  }

  return (
    <Alert>
      <HugeiconsIcon icon={isTechnicalAbort ? Bug01Icon : Agreement01Icon} />
      <AlertTitle>
        {isTechnicalAbort ? "Technical abort requested" : "End game by agreement?"}
      </AlertTitle>
      <AlertDescription>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex min-w-0 flex-col gap-1">
            <P>
              {proposer?.name ?? "A player"}{" "}
              {isTechnicalAbort
                ? "reported a game-breaking problem"
                : "would like to end without a winner"}
              .
            </P>
            {proposal.description ? <P className="truncate">“{proposal.description}”</P> : null}
          </div>
          <div className="flex shrink-0 items-center gap-2">
            <Badge variant="outline">
              <AnimatedNumber value={proposal.agreedPlayerIds.length} />/
              <AnimatedNumber value={proposal.eligiblePlayerIds.length} /> agreed
            </Badge>
            {hasAgreed ? (
              <Badge variant="secondary">Waiting for others</Badge>
            ) : (
              <>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  disabled={isVoting}
                  onClick={() => void vote(false)}
                >
                  Keep playing
                </Button>
                <Button type="button" size="sm" disabled={isVoting} onClick={() => void vote(true)}>
                  {isTechnicalAbort ? "Abort game" : "End game"}
                </Button>
              </>
            )}
          </div>
        </div>
      </AlertDescription>
    </Alert>
  );
}
