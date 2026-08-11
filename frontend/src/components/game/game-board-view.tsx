import {
  useEffect,
  useEffectEvent,
  useLayoutEffect,
  useRef,
  useState,
  type RefObject,
} from "react";
import {
  closestCenter,
  DndContext,
  DragOverlay,
  KeyboardSensor,
  MouseSensor,
  pointerWithin,
  TouchSensor,
  type CollisionDetection,
  type DragEndEvent,
  type DragOverEvent,
  type DragStartEvent,
  type Modifier,
  useSensor,
  useSensors,
} from "@dnd-kit/core";
import { sortableKeyboardCoordinates } from "@dnd-kit/sortable";
import {
  type ActionResult,
  type CardSnapshot,
  type DraftCompositionSnapshot,
  type GameSnapshot,
  type PlayerSnapshot,
  type SocialState,
  type TablePlayRequest,
  type TurnActivitySnapshot,
  useGameWebSocket,
} from "#/components/game-websocket-provider";
import { GameBoardHand } from "#/components/game/game-board-hand";
import { GameBoardPiles } from "#/components/game/game-board-piles";
import { GameBoardPlayers } from "#/components/game/game-board-players";
import { GameBoardTable } from "#/components/game/game-board-table";
import { GameCard } from "#/components/game/game-card";
import { CardTransferAnimation } from "#/components/game/card-transfer-animation";
import { CompletedCompositionAnimation } from "#/components/game/completed-composition-animation";
import { TurnStartCue } from "#/components/game/turn-start-cue";
import {
  setPersistedHandOrder,
  usePersistedHandOrder,
} from "#/components/game/game-hand-order-store";
import {
  FACE_DOWN_CARD,
  HAND_DROP_ID,
  NEW_COMPOSITION_DROP_ID,
  type ActiveDrag,
  type DraftComposition,
  type DraftedCompositionView,
  applyHandEntryOrder,
  buildTablePlayRequest,
  buildTableCompositionViews,
  buildHandEntries,
  buildDraftCompositionSnapshot,
  compositionIdFromDropId,
  findNewHandEntry,
  handIndexAfterSubmittedDrafts,
  handEntryOrder,
  insertHandKeyIntoDraft,
  mapHandKeysToEntries,
  moveHandEntry,
  moveDraftCompositionInsertIndex,
  removeHandKeyFromDrafts,
  resolveDraftViews,
  setDraftCardInsertIndex,
  setDraftCardReclaimTarget,
  tableCompositionEdgeTargetFromDropId,
  tableCompositionInsertIndexForEdge,
  tableCompositionJokerTargetFromDropId,
  validateDraftedCompositions,
  validateOpeningTablePlay,
} from "#/components/game/game-board-view-state";
import { playGameSound } from "#/lib/game-sounds";

const CARD_OVERLAY_VIEWPORT_MARGIN = 8;

const keepCardOverlayInViewport: Modifier = ({ overlayNodeRect, transform, windowRect }) => {
  if (!overlayNodeRect || !windowRect) {
    return transform;
  }

  const minX = windowRect.left + CARD_OVERLAY_VIEWPORT_MARGIN - overlayNodeRect.left;
  const maxX = windowRect.right - CARD_OVERLAY_VIEWPORT_MARGIN - overlayNodeRect.right;
  const minY = windowRect.top + CARD_OVERLAY_VIEWPORT_MARGIN - overlayNodeRect.top;
  const maxY = windowRect.bottom - CARD_OVERLAY_VIEWPORT_MARGIN - overlayNodeRect.bottom;

  return {
    ...transform,
    x: Math.min(Math.max(transform.x, minX), maxX),
    y: Math.min(Math.max(transform.y, minY), maxY),
  };
};

type GameBoardTurnState = {
  canDrawDeck: boolean;
  canDrawDiscard: boolean;
  canDiscard: boolean;
  isMyTurn: boolean;
  turnPlayerName: string;
};

type HandOrderState = {
  scopeKey: string | null;
  order: string[] | null;
};

type DraftCompositionState = {
  scopeKey: string | null;
  compositions: DraftComposition[];
  baselineHandOrder: string[] | null;
};

function createBoardCollisionDetection(handCardIds: Set<string>): CollisionDetection {
  return ({ droppableContainers, ...args }) => {
    const handCardContainers = droppableContainers.filter((container) =>
      handCardIds.has(String(container.id)),
    );

    const pointerCollisions = pointerWithin({
      ...args,
      droppableContainers,
    });
    const pointerSpecificCollisions = pointerCollisions.filter(
      (collision) => collision.id !== NEW_COMPOSITION_DROP_ID && collision.id !== HAND_DROP_ID,
    );
    const jokerSpecificCollisions = pointerSpecificCollisions.filter((collision) =>
      String(collision.id).startsWith("table-composition-joker-"),
    );
    const edgeSpecificCollisions = pointerSpecificCollisions.filter((collision) =>
      String(collision.id).startsWith("table-composition-edge-"),
    );

    if (jokerSpecificCollisions.length > 0) {
      return jokerSpecificCollisions;
    }

    if (edgeSpecificCollisions.length > 0) {
      return edgeSpecificCollisions;
    }

    if (pointerSpecificCollisions.length > 0) {
      return pointerSpecificCollisions;
    }

    if (
      pointerCollisions.some((collision) => collision.id === HAND_DROP_ID) &&
      handCardContainers.length > 0
    ) {
      const closestHandCollisions = closestCenter({
        ...args,
        droppableContainers: handCardContainers,
      });

      if (closestHandCollisions.length > 0) {
        return closestHandCollisions;
      }
    }

    if (pointerCollisions.length > 0) {
      return pointerCollisions;
    }

    return [];
  };
}

type GameBoardViewProps = {
  game: GameSnapshot | null;
  roomCode: string | null;
  playerId: string;
  players: PlayerSnapshot[];
  connectedPlayers: number;
  spectatorCount?: number;
  viewerMode?: "player" | "spectator";
  turnState: GameBoardTurnState;
  topDiscardCard: GameSnapshot["discardPile"][number] | null;
  onDiscardCard: (cardIndex: number, card: CardSnapshot) => Promise<ActionResult> | void;
  onDrawFromDeck: () => void;
  onDrawFromDiscard: () => void;
  onPlayTable: (play: TablePlayRequest) => Promise<ActionResult> | void;
  onPlayTableAndDiscard: (
    play: TablePlayRequest,
    cardIndex: number,
    card: CardSnapshot,
  ) => Promise<ActionResult> | void;
  onSendEmote: (emoji: string) => void;
  draftSyncMode?: "enabled" | "disabled";
  social?: SocialState;
  onSendFriendRequest?: (userId: string) => Promise<unknown>;
  guidance?: {
    stage: "orientation" | "draw" | "compose" | "discard";
    onDrawSettled: () => void;
    onDrawDragStateChange: (isDragging: boolean) => void;
    onDraftStateChange: (state: {
      hasDraftedCompositions: boolean;
      canSubmitTablePlay: boolean;
      draftCompositions: DraftCompositionSnapshot[];
      isDraggingCard: boolean;
    }) => void;
  };
};

type SubmittedCardActivity = {
  kind: "new" | "addition" | "joker_reclaim";
  playerId: string;
};

type SubmittedCompositionActivity = {
  kind?: string;
  playerId?: string;
  cardActivities: Record<number, SubmittedCardActivity>;
};

function draftScopeKey(game: GameSnapshot | null, playerId: string, isMyTurn: boolean) {
  if (!game || !isMyTurn) {
    return null;
  }

  return `${playerId}:${game.round}:${game.turn.number}`;
}

function buildDraftedCompositionViews(
  draftCompositions: DraftComposition[],
  entryByKey: Map<string, ReturnType<typeof buildHandEntries>[number]>,
) {
  return draftCompositions.map((composition) => ({
    ...composition,
    entries: mapHandKeysToEntries(composition.handKeys, entryByKey),
  })) as DraftedCompositionView[];
}

function cardEquals(a: GameSnapshot["hand"][number], b: GameSnapshot["hand"][number]) {
  return Boolean(a.isJoker) === Boolean(b.isJoker) && a.rank === b.rank && a.suit === b.suit;
}

function buildSubmittedCompositionActivityMap(
  baselineCompositions: GameSnapshot["activeCompositions"],
  currentCompositions: GameSnapshot["activeCompositions"],
  turnActivity?: TurnActivitySnapshot,
) {
  const activityMap = new Map<number, SubmittedCompositionActivity>();
  const compositionActivityByIndex = new Map(
    (turnActivity?.compositionActivities ?? []).map((compositionActivity) => [
      compositionActivity.tableIndex,
      compositionActivity,
    ]),
  );

  for (const compositionActivity of turnActivity?.compositionActivities ?? []) {
    activityMap.set(compositionActivity.tableIndex, {
      kind: compositionActivity.kind,
      playerId: compositionActivity.playerId,
      cardActivities: {},
    });
  }

  for (let index = 0; index < currentCompositions.length; index += 1) {
    const current = currentCompositions[index];
    const baseline = baselineCompositions[index];
    const activity = activityMap.get(index) ?? {
      kind: undefined,
      playerId: undefined,
      cardActivities: {},
    };

    if (!baseline) {
      activity.kind ??= "new_composition";
      for (let cardIndex = 0; cardIndex < current.cards.length; cardIndex += 1) {
        const activityDetails = compositionActivityByIndex.get(index)?.cardActivities?.[cardIndex];
        if (activityDetails) {
          activity.cardActivities[cardIndex] = {
            kind: activityDetails.kind === "joker_reclaim" ? "joker_reclaim" : "new",
            playerId: activityDetails.playerId,
          };
        }
      }
      activityMap.set(index, activity);
      continue;
    }

    for (let cardIndex = 0; cardIndex < current.cards.length; cardIndex += 1) {
      const currentCard = current.cards[cardIndex];
      const baselineCard = baseline.cards[cardIndex];
      const activityDetails = compositionActivityByIndex.get(index)?.cardActivities?.[cardIndex];

      if (!baselineCard) {
        if (activityDetails) {
          activity.cardActivities[cardIndex] = {
            kind: activityDetails.kind === "joker_reclaim" ? "joker_reclaim" : "addition",
            playerId: activityDetails.playerId,
          };
          activityMap.set(index, activity);
        }
        continue;
      }

      if (
        activityDetails &&
        (activityDetails.kind === "joker_reclaim" || !cardEquals(currentCard, baselineCard))
      ) {
        activity.cardActivities[cardIndex] = {
          kind: activityDetails.kind === "joker_reclaim" ? "joker_reclaim" : "addition",
          playerId: activityDetails.playerId,
        };
        activityMap.set(index, activity);
      }
    }
  }

  return activityMap;
}

function GameBoardLayout({
  boardRef,
  game,
  tableCompositions,
  newCompositions,
  submittedCompositionActivities,
  turnState,
  topDiscardCard,
  players,
  connectedPlayers,
  spectatorCount,
  availableHandEntries,
  sortableIds,
  activeDrag,
  draftedCompositionsCount,
  spectatorDrafts,
  invalidCompositionIds,
  invalidEntryKeys,
  onResetDraftCompositions,
  onSendEmote,
  currentPlayerId,
  social,
  onSendFriendRequest,
  viewerMode,
  guidanceStage,
}: {
  boardRef: RefObject<HTMLDivElement | null>;
  game: GameSnapshot | null;
  tableCompositions: ReturnType<typeof buildTableCompositionViews>;
  newCompositions: DraftedCompositionView[];
  submittedCompositionActivities: Map<number, SubmittedCompositionActivity>;
  turnState: GameBoardTurnState;
  topDiscardCard: GameSnapshot["discardPile"][number] | null;
  players: PlayerSnapshot[];
  connectedPlayers: number;
  spectatorCount: number;
  viewerMode: "player" | "spectator";
  availableHandEntries: ReturnType<typeof buildHandEntries>;
  sortableIds: string[];
  activeDrag: ActiveDrag | null;
  draftedCompositionsCount: number;
  spectatorDrafts: DraftCompositionSnapshot[];
  invalidCompositionIds: Set<string>;
  invalidEntryKeys: Set<string>;
  onResetDraftCompositions: () => void;
  onSendEmote: (emoji: string) => void;
  currentPlayerId: string;
  social?: SocialState;
  onSendFriendRequest?: (userId: string) => Promise<unknown>;
  guidanceStage?: "orientation" | "draw" | "compose" | "discard";
}) {
  const currentPlayer = players.find((player) => player.playerId === currentPlayerId);
  const isSpectating = viewerMode === "spectator";

  return (
    <div
      ref={boardRef}
      className="grid min-h-0 min-w-0 flex-1 grid-cols-1 grid-rows-[auto_minmax(8rem,1fr)_auto_auto] gap-2 p-1 xl:grid-cols-[minmax(0,1fr)_22rem] xl:grid-rows-[minmax(0,1fr)_auto_auto] xl:gap-4 [@media(max-height:600px)]:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)] [@media(max-height:600px)]:grid-rows-[auto_minmax(0,1fr)_5.25rem] [@media(max-height:600px)]:gap-2"
    >
      <div className="col-start-1 row-start-1 min-w-0 xl:col-start-2 xl:row-start-1 xl:flex xl:min-h-0 [@media(max-height:600px)]:col-span-2 [@media(max-height:600px)]:col-start-1 [@media(max-height:600px)]:row-start-1">
        <GameBoardPlayers
          players={players}
          game={game}
          connectedPlayers={connectedPlayers}
          spectatorCount={spectatorCount}
          canSendEmote={!isSpectating}
          hasDraftedCompositions={draftedCompositionsCount > 0}
          compactOnMobile
          onResetDraftCompositions={onResetDraftCompositions}
          onSendEmote={onSendEmote}
          currentPlayerId={currentPlayerId}
          social={social}
          onSendFriendRequest={onSendFriendRequest}
        />
      </div>

      <div
        data-slot="game-board-table-region"
        className="relative col-start-1 row-start-2 min-h-0 min-w-0 xl:col-start-1 xl:row-span-2 xl:row-start-1 [@media(max-height:600px)]:col-start-1 [@media(max-height:600px)]:row-span-2 [@media(max-height:600px)]:row-start-2"
      >
        {!guidanceStage && turnState.isMyTurn && game ? (
          <TurnStartCue
            key={`${game.round}:${game.turn.number}`}
            round={game.round}
            turnNumber={game.turn.number}
            playerName={currentPlayer?.name ?? turnState.turnPlayerName}
            playerImageUrl={currentPlayer?.imageUrl}
          />
        ) : null}

        <GameBoardTable
          tableCompositions={tableCompositions}
          newCompositions={newCompositions}
          players={players}
          turnActivity={
            game?.turnActivity
              ? {
                  ...game.turnActivity,
                  draftCompositions: spectatorDrafts,
                  compositionActivities: Array.from(submittedCompositionActivities.entries()).map(
                    ([tableIndex, activity]) => ({
                      tableIndex,
                      kind: activity.kind,
                      playerId: activity.playerId,
                      cardActivities: activity.cardActivities,
                    }),
                  ),
                }
              : undefined
          }
          canCompose={turnState.canDiscard}
          viewerPlayerId={currentPlayerId}
          showDraftTotal={
            !game?.players.find((player) => player.playerId === game.turn.playerId)?.hasOpened
          }
          invalidCompositionIds={invalidCompositionIds}
          invalidEntryKeys={invalidEntryKeys}
          showNewCompositionDropCue={guidanceStage === "compose"}
        />
      </div>

      <div className="col-start-1 row-start-4 min-h-0 min-w-0 xl:col-start-2 xl:row-start-2 [@media(max-height:600px)]:col-start-2 [@media(max-height:600px)]:row-start-3">
        <GameBoardPiles
          drawPileCount={game?.drawPileCount ?? 0}
          discardPileCount={game?.discardPile.length ?? 0}
          topDiscardCard={topDiscardCard}
          canDrawDeck={turnState.canDrawDeck}
          canDrawDiscard={turnState.canDrawDiscard}
          canDiscard={turnState.canDiscard}
        />
      </div>

      <div className="col-start-1 row-start-3 grid min-h-0 min-w-0 shrink-0 xl:col-span-2 xl:row-start-3 [@media(max-height:600px)]:col-span-1 [@media(max-height:600px)]:col-start-2 [@media(max-height:600px)]:row-start-2">
        <GameBoardHand
          status={{
            hasGame: Boolean(game),
            isMyTurn: turnState.isMyTurn,
            turnPlayerName: turnState.turnPlayerName,
            isSpectating,
          }}
          availableHandEntries={availableHandEntries}
          sortableIds={sortableIds}
          activeDrag={activeDrag}
          tablePlayState={{
            hasDraftedCompositions: draftedCompositionsCount > 0,
          }}
        />
      </div>
    </div>
  );
}

type GameBoardController = {
  activeDrag: ActiveDrag | null;
  activeEntry: ReturnType<typeof buildHandEntries>[number] | null;
  availableHandEntries: ReturnType<typeof buildHandEntries>;
  sortableIds: string[];
  tableCompositions: ReturnType<typeof buildTableCompositionViews>;
  newCompositions: DraftedCompositionView[];
  draftedCompositionsCount: number;
  invalidCompositionIds: Set<string>;
  invalidEntryKeys: Set<string>;
  handleDragStart: (event: DragStartEvent) => void;
  handleDragOver: (event: DragOverEvent) => void;
  handleDragEnd: (event: DragEndEvent) => void;
  handleDragCancel: () => void;
  resetDraftCompositions: () => void;
  submitDraftCompositions: () => Promise<ActionResult | void>;
};

function useGameBoardController({
  game,
  roomCode,
  playerId,
  turnState,
  topDiscardCard,
  onDiscardCard,
  onDrawFromDeck,
  onDrawFromDiscard,
  onPlayTable,
  onPlayTableAndDiscard,
  draftSyncMode = "enabled",
  guidance,
}: Pick<
  GameBoardViewProps,
  | "game"
  | "roomCode"
  | "playerId"
  | "turnState"
  | "topDiscardCard"
  | "onDiscardCard"
  | "onDrawFromDeck"
  | "onDrawFromDiscard"
  | "onPlayTable"
  | "onPlayTableAndDiscard"
  | "draftSyncMode"
  | "guidance"
>): GameBoardController {
  const { updateTurnDrafts } = useGameWebSocket();
  const handOrderScopeKey = roomCode && playerId ? `${roomCode}:${playerId}` : null;
  const persistedHandOrder = usePersistedHandOrder(handOrderScopeKey);
  const [handOrderState, setHandOrderState] = useState<HandOrderState>({
    scopeKey: null,
    order: null,
  });
  const [activeDrag, setActiveDrag] = useState<ActiveDrag | null>(null);
  const [draftCompositionState, setDraftCompositionState] = useState<DraftCompositionState>({
    scopeKey: null,
    compositions: [],
    baselineHandOrder: null,
  });
  const draftCompositionStateRef = useRef(draftCompositionState);
  const [showDraftValidation, setShowDraftValidation] = useState(false);
  const nextDraftIdRef = useRef(0);
  const skipNextDraftSyncRef = useRef(false);
  const rawHandEntries = buildHandEntries(game?.hand ?? []);
  const handOrder =
    handOrderState.scopeKey === handOrderScopeKey && handOrderState.order
      ? handOrderState.order
      : persistedHandOrder;
  const handEntries = applyHandEntryOrder(rawHandEntries, handOrder);
  const currentDraftScopeKey = draftScopeKey(game, playerId, turnState.isMyTurn);

  useLayoutEffect(() => {
    draftCompositionStateRef.current = draftCompositionState;
  }, [draftCompositionState]);

  function commitDraftCompositionState(nextState: DraftCompositionState) {
    draftCompositionStateRef.current = nextState;
    setDraftCompositionState(nextState);
  }

  function updateHandOrder(order: string[]) {
    setHandOrderState({
      scopeKey: handOrderScopeKey,
      order,
    });
    setPersistedHandOrder(handOrderScopeKey, order);
  }

  const activeDrawEntry =
    activeDrag?.type === "draw"
      ? findNewHandEntry(
          activeDrag.baselineEntries,
          applyHandEntryOrder(rawHandEntries, activeDrag.baselineOrder),
        )
      : null;

  function reportGuidanceDraftState(nextCompositions: DraftComposition[], isDraggingCard: boolean) {
    if (!guidance) return;

    const nextResolution = resolveDraftViews(
      game?.activeCompositions ?? [],
      nextCompositions,
      handEntries,
    );
    const nextViews = buildDraftedCompositionViews(
      nextResolution.draftCompositions,
      nextResolution.allEntryByKey,
    );
    const playerHasOpened =
      game?.players.find((player) => player.playerId === playerId)?.hasOpened ?? false;
    const nextOpeningValidation = validateOpeningTablePlay(playerHasOpened, nextViews);
    guidance.onDraftStateChange({
      hasDraftedCompositions: nextViews.length > 0,
      canSubmitTablePlay:
        turnState.canDiscard && nextViews.length > 0 && nextOpeningValidation.canSubmit,
      draftCompositions: nextViews.map(buildDraftCompositionSnapshot),
      isDraggingCard,
    });
  }

  function updateDraftCompositions(
    updater: (current: DraftComposition[]) => DraftComposition[],
    isDraggingCard: boolean,
  ) {
    setShowDraftValidation(false);
    const current = draftCompositionStateRef.current;
    const isCurrentScope = current.scopeKey === currentDraftScopeKey;
    const scopedCompositions = isCurrentScope ? current.compositions : [];
    const nextCompositions = updater(scopedCompositions);

    commitDraftCompositionState({
      scopeKey: currentDraftScopeKey,
      compositions: nextCompositions,
      baselineHandOrder:
        (isCurrentScope ? current.baselineHandOrder : null) ??
        (scopedCompositions.length === 0 && nextCompositions.length > 0
          ? handEntryOrder(handEntries)
          : null),
    });
    reportGuidanceDraftState(nextCompositions, isDraggingCard);
  }

  function finishReturningHandKey(handKey: string) {
    const current = draftCompositionStateRef.current;
    const scopedCompositions =
      current.scopeKey === currentDraftScopeKey ? current.compositions : [];
    const nextCompositions = removeHandKeyFromDrafts(scopedCompositions, handKey);

    commitDraftCompositionState({
      scopeKey: currentDraftScopeKey,
      compositions: nextCompositions,
      baselineHandOrder: nextCompositions.length > 0 ? current.baselineHandOrder : null,
    });
    reportGuidanceDraftState(nextCompositions, false);
  }

  const scopedDraftCompositions =
    draftCompositionState.scopeKey === currentDraftScopeKey
      ? draftCompositionState.compositions
      : [];
  const draftResolution = resolveDraftViews(
    game?.activeCompositions ?? [],
    scopedDraftCompositions,
    handEntries,
  );
  const tableCompositions = draftResolution.tableCompositions;
  const allHandEntries = draftResolution.allHandEntries;
  const draftCompositions = draftResolution.draftCompositions;
  const activeEntry =
    activeDrag?.type === "hand"
      ? (allHandEntries.find((entry) => entry.key === activeDrag.handKey) ?? null)
      : null;
  const allEntryByKey = draftResolution.allEntryByKey;
  const stagedHandKeySet = new Set(
    draftCompositions.flatMap((composition) => composition.handKeys),
  );
  const availableHandEntries = allHandEntries.filter((entry) => !stagedHandKeySet.has(entry.key));
  const sortableIds = availableHandEntries.map((entry) => entry.key);
  const draftedCompositionsView = buildDraftedCompositionViews(draftCompositions, allEntryByKey);
  const newCompositions = draftedCompositionsView.filter(
    (composition) => composition.tableIndex === null,
  );
  const canCompose = turnState.canDiscard;
  const currentPlayerHasOpened =
    game?.players.find((player) => player.playerId === playerId)?.hasOpened ?? false;
  const openingTablePlayValidation = validateOpeningTablePlay(
    currentPlayerHasOpened,
    draftedCompositionsView,
  );
  const canSubmitTablePlay =
    canCompose && draftedCompositionsView.length > 0 && openingTablePlayValidation.canSubmit;
  const draftValidation = showDraftValidation
    ? validateDraftedCompositions(game?.activeCompositions ?? [], draftedCompositionsView)
    : { invalidCompositionIds: new Set<string>(), invalidEntryKeys: new Set<string>() };

  const serializedDrafts = JSON.stringify(
    draftedCompositionsView.map(buildDraftCompositionSnapshot),
  );

  const syncTurnDrafts = useEffectEvent((drafts: DraftCompositionSnapshot[]) => {
    updateTurnDrafts({ compositions: drafts });
  });

  useEffect(() => {
    if (draftSyncMode === "disabled" || !turnState.isMyTurn) {
      return;
    }

    if (skipNextDraftSyncRef.current) {
      skipNextDraftSyncRef.current = false;
      return;
    }

    syncTurnDrafts(JSON.parse(serializedDrafts) as DraftCompositionSnapshot[]);
  }, [draftSyncMode, serializedDrafts, turnState.isMyTurn]);

  function resetDraftCompositions({ sync = true }: { sync?: boolean } = {}) {
    setShowDraftValidation(false);
    if (!sync) {
      skipNextDraftSyncRef.current = true;
    }

    const current = draftCompositionStateRef.current;
    if (current.scopeKey === currentDraftScopeKey && current.baselineHandOrder) {
      updateHandOrder(current.baselineHandOrder);
    }

    commitDraftCompositionState({
      scopeKey: currentDraftScopeKey,
      compositions: [],
      baselineHandOrder: null,
    });
  }

  async function submitDraftCompositions() {
    if (!canSubmitTablePlay) {
      return { action: "play_table", playerId, ok: false };
    }

    let result: ActionResult | void;
    try {
      result = await onPlayTable(
        buildTablePlayRequest(game?.activeCompositions ?? [], draftedCompositionsView),
      );
    } catch {
      setShowDraftValidation(true);
      playGameSound("invalid-action");
      return { action: "play_table", playerId, ok: false };
    }

    if (result?.ok === false) {
      setShowDraftValidation(true);
      playGameSound("invalid-action");
      return result;
    }

    resetDraftCompositions({ sync: false });
    return result;
  }

  function handleDragStart(event: DragStartEvent) {
    const drawSource = event.active.data.current?.drawSource;

    if (drawSource === "deck") {
      playGameSound("card-pickup");
      guidance?.onDrawDragStateChange(true);
      setActiveDrag({
        type: "draw",
        source: "deck",
        card: null,
        baselineEntries: handEntries,
        baselineOrder: handEntryOrder(handEntries),
        revealedHandKey: null,
      });
      onDrawFromDeck();
      return;
    }

    if (drawSource === "discard" && topDiscardCard) {
      playGameSound("card-pickup");
      guidance?.onDrawDragStateChange(true);
      setActiveDrag({
        type: "draw",
        source: "discard",
        card: topDiscardCard,
        baselineEntries: handEntries,
        baselineOrder: handEntryOrder(handEntries),
        revealedHandKey: null,
      });
      onDrawFromDiscard();
      return;
    }

    const handKey = typeof event.active.id === "string" ? event.active.id : null;
    if (handKey) {
      playGameSound("card-pickup");
    }
    setActiveDrag(
      handKey
        ? {
            type: "hand",
            handKey,
            baselineHandOrder: handEntryOrder(handEntries),
            baselineDraftCompositions: draftCompositions,
            hasReorderedHand: false,
          }
        : null,
    );
  }

  function handleDragOver(event: DragOverEvent) {
    if (activeDrag?.type === "draw") {
      if (activeDrawEntry === null) {
        return;
      }

      const overHandKey = typeof event.over?.id === "string" ? event.over.id : null;

      if (overHandKey === null || overHandKey === "discard-pile") {
        updateHandOrder(activeDrag.baselineOrder);
        return;
      }

      updateHandOrder(handEntryOrder(moveHandEntry(handEntries, activeDrawEntry.key, overHandKey)));
      return;
    }

    if (activeDrag?.type !== "hand") {
      return;
    }

    const overId = typeof event.over?.id === "string" ? event.over.id : null;
    if (overId === null) {
      return;
    }

    const startedInDraft = activeDrag.baselineDraftCompositions.some((composition) =>
      composition.handKeys.includes(activeDrag.handKey),
    );
    const droppedOnHandCard = availableHandEntries.some((entry) => entry.key === overId);

    if (!startedInDraft && droppedOnHandCard && activeDrag.handKey !== overId) {
      updateHandOrder(handEntryOrder(moveHandEntry(handEntries, activeDrag.handKey, overId)));
      setActiveDrag((current) =>
        current?.type === "hand" ? { ...current, hasReorderedHand: true } : current,
      );
      return;
    }

    if (!canCompose) {
      return;
    }

    if (startedInDraft && (overId === HAND_DROP_ID || droppedOnHandCard)) {
      updateDraftCompositions(
        (current) => removeHandKeyFromDrafts(current, activeDrag.handKey),
        true,
      );

      if (droppedOnHandCard && activeDrag.handKey !== overId) {
        updateHandOrder(handEntryOrder(moveHandEntry(handEntries, activeDrag.handKey, overId)));
      }
      return;
    }

    const droppedOnDraftCard = draftCompositions.find((composition) =>
      composition.handKeys.includes(overId),
    );
    if (droppedOnDraftCard?.tableIndex === null) {
      updateDraftCompositions(
        (current) =>
          insertHandKeyIntoDraft(current, activeDrag.handKey, droppedOnDraftCard.id, overId),
        true,
      );
      return;
    }

    const droppedOnDraftContainer = compositionIdFromDropId(overId);
    if (!droppedOnDraftContainer) {
      return;
    }

    const target = draftCompositions.find(
      (composition) => composition.id === droppedOnDraftContainer,
    );
    if (target?.tableIndex !== null) {
      return;
    }

    updateDraftCompositions(
      (current) => insertHandKeyIntoDraft(current, activeDrag.handKey, droppedOnDraftContainer),
      true,
    );
  }

  function handleDragEnd(event: DragEndEvent) {
    const currentDrag = activeDrag;
    const draggedHandKey = typeof event.active.id === "string" ? event.active.id : null;
    const overHandKey = typeof event.over?.id === "string" ? event.over.id : null;
    const overId = typeof event.over?.id === "string" ? event.over.id : null;

    setActiveDrag(null);

    if (currentDrag?.type === "draw") {
      guidance?.onDrawDragStateChange(false);
      guidance?.onDrawSettled();
      if (activeDrawEntry?.key && overHandKey && overHandKey !== "discard-pile") {
        return;
      }

      updateHandOrder(currentDrag.baselineOrder);
      return;
    }

    if (currentDrag?.type !== "hand" || typeof draggedHandKey !== "string") {
      return;
    }

    if (overId === null) {
      updateHandOrder(currentDrag.baselineHandOrder);
      commitDraftCompositionState({
        scopeKey: currentDraftScopeKey,
        compositions: currentDrag.baselineDraftCompositions,
        baselineHandOrder: draftCompositionStateRef.current.baselineHandOrder,
      });
      reportGuidanceDraftState(currentDrag.baselineDraftCompositions, false);
      return;
    }

    const draggedEntry = allEntryByKey.get(draggedHandKey) ?? null;
    const droppedOnDraftCard =
      overId === null
        ? null
        : draftCompositions.find((composition) => composition.handKeys.includes(overId));
    const droppedOnHandCard =
      overId !== null && availableHandEntries.some((entry) => entry.key === overId);
    const startedInDraft = currentDrag.baselineDraftCompositions.some((composition) =>
      composition.handKeys.includes(draggedHandKey),
    );
    const returnedToHandDuringDrag =
      startedInDraft &&
      !draftCompositions.some((composition) => composition.handKeys.includes(draggedHandKey));
    const droppedOnDraftContainer = overId ? compositionIdFromDropId(overId) : null;
    const droppedOnTableEdgeTarget = overId ? tableCompositionEdgeTargetFromDropId(overId) : null;
    const droppedOnTableJokerTarget = overId ? tableCompositionJokerTargetFromDropId(overId) : null;

    if (event.over?.id === "discard-pile") {
      if (draftCompositions.some((composition) => composition.handKeys.includes(draggedHandKey))) {
        return;
      }

      const cardIndex = event.active.data.current?.cardIndex;
      const isVirtual = event.active.data.current?.isVirtual === true;
      if (isVirtual) {
        playGameSound("invalid-action");
        return;
      }
      if (typeof cardIndex === "number") {
        void (async () => {
          const discardIndex =
            draftedCompositionsView.length > 0
              ? handIndexAfterSubmittedDrafts(
                  rawHandEntries,
                  draggedHandKey,
                  draftedCompositionsView,
                )
              : cardIndex;

          if (discardIndex === null) {
            return;
          }

          const discardCard = draggedEntry?.card;
          if (!discardCard) {
            return;
          }

          if (draftedCompositionsView.length > 0) {
            try {
              const result = await onPlayTableAndDiscard(
                buildTablePlayRequest(game?.activeCompositions ?? [], draftedCompositionsView),
                discardIndex,
                discardCard,
              );
              if (result?.ok === false) {
                setShowDraftValidation(true);
                playGameSound("invalid-action");
              }
            } catch {
              setShowDraftValidation(true);
              playGameSound("invalid-action");
            }
            return;
          }

          await onDiscardCard(discardIndex, discardCard);
        })();
      }
      return;
    }

    if (!canCompose) {
      if (droppedOnHandCard && draggedHandKey !== overId && !currentDrag.hasReorderedHand) {
        updateHandOrder(handEntryOrder(moveHandEntry(handEntries, draggedHandKey, overId)));
        playGameSound("card-place");
      }
      return;
    }

    if (droppedOnTableEdgeTarget !== null && draggedEntry) {
      playGameSound("card-place");
      const newCompositionId = `draft-${nextDraftIdRef.current}`;
      nextDraftIdRef.current += 1;
      updateDraftCompositions((current) => {
        const existing = current.find(
          (composition) => composition.tableIndex === droppedOnTableEdgeTarget.compositionIndex,
        );
        const targetComposition =
          game?.activeCompositions?.[droppedOnTableEdgeTarget.compositionIndex];
        const insertIndex = targetComposition
          ? tableCompositionInsertIndexForEdge(targetComposition, droppedOnTableEdgeTarget.edge)
          : 0;

        if (existing) {
          return setDraftCardInsertIndex(
            insertHandKeyIntoDraft(current, draggedHandKey, existing.id),
            existing.id,
            draggedHandKey,
            insertIndex,
          );
        }

        return [
          ...removeHandKeyFromDrafts(current, draggedHandKey),
          {
            id: newCompositionId,
            handKeys: [draggedHandKey],
            tableIndex: droppedOnTableEdgeTarget.compositionIndex,
            insertIndex,
            cardInsertIndices: { [draggedHandKey]: insertIndex },
          },
        ];
      }, false);
      return;
    }

    if (droppedOnTableJokerTarget !== null && draggedEntry) {
      playGameSound("card-place");
      const newCompositionId = `draft-${nextDraftIdRef.current}`;
      nextDraftIdRef.current += 1;
      updateDraftCompositions((current) => {
        const existing = current.find(
          (composition) => composition.tableIndex === droppedOnTableJokerTarget.compositionIndex,
        );

        if (existing) {
          return setDraftCardReclaimTarget(
            insertHandKeyIntoDraft(current, draggedHandKey, existing.id),
            existing.id,
            draggedHandKey,
            droppedOnTableJokerTarget.jokerIndex,
          );
        }

        return [
          ...removeHandKeyFromDrafts(current, draggedHandKey),
          {
            id: newCompositionId,
            handKeys: [draggedHandKey],
            tableIndex: droppedOnTableJokerTarget.compositionIndex,
            insertIndex:
              game?.activeCompositions?.[droppedOnTableJokerTarget.compositionIndex]?.cards
                .length ?? 0,
            reclaimTargets: { [draggedHandKey]: droppedOnTableJokerTarget.jokerIndex },
          },
        ];
      }, false);
      return;
    }

    if (overId === NEW_COMPOSITION_DROP_ID) {
      playGameSound("card-place");
      const compositionId = `draft-${nextDraftIdRef.current}`;
      nextDraftIdRef.current += 1;

      updateDraftCompositions(
        (current) => [
          ...removeHandKeyFromDrafts(current, draggedHandKey),
          { id: compositionId, handKeys: [draggedHandKey], tableIndex: null },
        ],
        false,
      );
      return;
    }

    if (droppedOnDraftCard) {
      playGameSound("card-place");
      updateDraftCompositions(
        (current) =>
          insertHandKeyIntoDraft(
            current,
            draggedHandKey,
            droppedOnDraftCard.id,
            overId ?? undefined,
          ),
        false,
      );
      return;
    }

    if (droppedOnDraftContainer) {
      playGameSound("card-place");
      updateDraftCompositions((current) => {
        const target = current.find((composition) => composition.id === droppedOnDraftContainer);
        const next = insertHandKeyIntoDraft(current, draggedHandKey, droppedOnDraftContainer);

        if (target && target.tableIndex !== null) {
          return moveDraftCompositionInsertIndex(
            next,
            droppedOnDraftContainer,
            target.insertIndex ?? target.handKeys.length,
          );
        }

        return next;
      }, false);
      return;
    }

    if (overId === HAND_DROP_ID) {
      playGameSound("card-place");
      finishReturningHandKey(draggedHandKey);
      return;
    }

    if (droppedOnHandCard) {
      playGameSound("card-place");
      finishReturningHandKey(draggedHandKey);

      if (draggedHandKey !== overId && !returnedToHandDuringDrag && !currentDrag.hasReorderedHand) {
        updateHandOrder(handEntryOrder(moveHandEntry(handEntries, draggedHandKey, overId)));
      }
    }
  }

  function handleDragCancel() {
    if (activeDrag?.type === "draw") {
      guidance?.onDrawDragStateChange(false);
      guidance?.onDrawSettled();
      updateHandOrder(activeDrag.baselineOrder);
    } else if (activeDrag?.type === "hand") {
      updateHandOrder(activeDrag.baselineHandOrder);
      commitDraftCompositionState({
        scopeKey: currentDraftScopeKey,
        compositions: activeDrag.baselineDraftCompositions,
        baselineHandOrder: draftCompositionStateRef.current.baselineHandOrder,
      });
      reportGuidanceDraftState(activeDrag.baselineDraftCompositions, false);
    }

    setActiveDrag(null);
  }

  const displayActiveDrag =
    activeDrag?.type === "draw"
      ? { ...activeDrag, revealedHandKey: activeDrawEntry?.key ?? null }
      : activeDrag;

  return {
    activeDrag: displayActiveDrag,
    activeEntry,
    availableHandEntries,
    sortableIds,
    tableCompositions,
    newCompositions,
    draftedCompositionsCount: draftedCompositionsView.length,
    invalidCompositionIds: draftValidation.invalidCompositionIds,
    invalidEntryKeys: draftValidation.invalidEntryKeys,
    handleDragStart,
    handleDragOver,
    handleDragEnd,
    handleDragCancel,
    resetDraftCompositions,
    submitDraftCompositions,
  };
}

export function GameBoardView({
  game,
  roomCode,
  playerId,
  players,
  connectedPlayers,
  spectatorCount = 0,
  viewerMode = "player",
  turnState,
  topDiscardCard,
  onDiscardCard,
  onDrawFromDeck,
  onDrawFromDiscard,
  onPlayTable,
  onPlayTableAndDiscard,
  onSendEmote,
  draftSyncMode = "enabled",
  social,
  onSendFriendRequest,
  guidance,
}: GameBoardViewProps) {
  const boardRef = useRef<HTMLDivElement>(null);
  const sensors = useSensors(
    useSensor(MouseSensor, {
      activationConstraint: { distance: 4 },
    }),
    useSensor(TouchSensor, {
      activationConstraint: {
        delay: 180,
        tolerance: 8,
      },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    }),
  );
  const controller = useGameBoardController({
    game,
    roomCode,
    playerId,
    turnState,
    topDiscardCard,
    onDiscardCard,
    onDrawFromDeck,
    onDrawFromDiscard,
    onPlayTable,
    onPlayTableAndDiscard,
    draftSyncMode,
    guidance,
  });
  const activeDraw = controller.activeDrag?.type === "draw" ? controller.activeDrag : null;
  const hasSubmittedTurnActivity = (game?.turnActivity?.compositionActivities?.length ?? 0) > 0;
  const submittedCompositionActivities = hasSubmittedTurnActivity
    ? new Map<number, SubmittedCompositionActivity>()
    : buildSubmittedCompositionActivityMap(
        game?.turnActivity?.baselineCompositions ?? game?.activeCompositions ?? [],
        game?.activeCompositions ?? [],
        game?.turnActivity,
      );
  const spectatorDrafts = hasSubmittedTurnActivity
    ? []
    : (game?.turnActivity?.draftCompositions ?? []);
  const collisionDetection = createBoardCollisionDetection(
    new Set(controller.availableHandEntries.map((entry) => entry.key)),
  );
  return (
    <DndContext
      sensors={sensors}
      collisionDetection={collisionDetection}
      onDragStart={controller.handleDragStart}
      onDragOver={controller.handleDragOver}
      onDragEnd={controller.handleDragEnd}
      onDragCancel={controller.handleDragCancel}
    >
      <GameBoardLayout
        boardRef={boardRef}
        game={game}
        tableCompositions={controller.tableCompositions}
        newCompositions={controller.newCompositions}
        submittedCompositionActivities={submittedCompositionActivities}
        turnState={turnState}
        topDiscardCard={topDiscardCard}
        players={players}
        connectedPlayers={connectedPlayers}
        spectatorCount={spectatorCount}
        viewerMode={viewerMode}
        availableHandEntries={controller.availableHandEntries}
        sortableIds={controller.sortableIds}
        activeDrag={controller.activeDrag}
        draftedCompositionsCount={controller.draftedCompositionsCount}
        spectatorDrafts={spectatorDrafts}
        invalidCompositionIds={controller.invalidCompositionIds}
        invalidEntryKeys={controller.invalidEntryKeys}
        onResetDraftCompositions={controller.resetDraftCompositions}
        onSendEmote={onSendEmote}
        currentPlayerId={playerId}
        social={social}
        onSendFriendRequest={onSendFriendRequest}
        guidanceStage={guidance?.stage}
      />
      <CardTransferAnimation boardRef={boardRef} game={game} viewerPlayerId={playerId} />
      <CompletedCompositionAnimation boardRef={boardRef} game={game} viewerPlayerId={playerId} />
      <DragOverlay dropAnimation={null} modifiers={[keepCardOverlayInViewport]}>
        {activeDraw ? (
          <GameCard
            card={
              (activeDraw.revealedHandKey
                ? controller.availableHandEntries.find(
                    (entry) => entry.key === activeDraw.revealedHandKey,
                  )?.card
                : null) ??
              activeDraw.card ??
              FACE_DOWN_CARD
            }
            faceDown={activeDraw.revealedHandKey === null && activeDraw.card === null}
            size="hand"
            className="rotate-3 opacity-75 shadow-xl ring-1 ring-foreground/10"
          />
        ) : controller.activeEntry ? (
          <GameCard
            card={controller.activeEntry.card}
            size="hand"
            className="rotate-3 opacity-75 shadow-xl ring-1 ring-foreground/10"
          />
        ) : null}
      </DragOverlay>
    </DndContext>
  );
}
