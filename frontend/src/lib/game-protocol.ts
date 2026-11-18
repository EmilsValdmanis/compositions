export type GameMode = "quick" | "full";

export type CardSnapshot = {
  rank?: number;
  suit?: number;
  isJoker?: boolean;
};

export type CompositionSnapshot = {
  type: string;
  cards: CardSnapshot[];
  jokerRepresentations?: Record<number, CardSnapshot[]>;
  points: number;
  complete: boolean;
};

export type CardActivitySnapshot = {
  kind: string;
  playerId: string;
};

export type CompositionActivitySnapshot = {
  tableIndex: number;
  kind?: string;
  playerId?: string;
  cardActivities?: Record<number, CardActivitySnapshot>;
};

export type DraftCompositionSnapshot = {
  id: string;
  tableIndex?: number;
  insertIndex?: number;
  cardPlacements?: Array<{
    insertIndex?: number;
    reclaimJokerIndex?: number;
  }>;
  cards: CardSnapshot[];
};

export type TurnActivitySnapshot = {
  playerId: string;
  round: number;
  turnNumber: number;
  drawSource?: "deck" | "discard";
  baselineCompositions?: CompositionSnapshot[];
  draftCompositions?: DraftCompositionSnapshot[];
  compositionActivities?: CompositionActivitySnapshot[];
};

export type CompositionDraftRequest = {
  cards: CardSnapshot[];
};

export type CompositionAdditionRequest = {
  compositionIndex: number;
  insertIndex?: number;
  cards: CardSnapshot[];
};

export type JokerReclaimRequest = {
  compositionIndex: number;
  jokerIndex: number;
  replacementCard: CardSnapshot;
};

export type TablePlayRequest = {
  compositions: CompositionDraftRequest[];
  additions: CompositionAdditionRequest[];
  reclaims: JokerReclaimRequest[];
};

export type DealingChoiceRequest = {
  dealType: string;
  cutSize: number;
  order?: number[];
};

export type TurnDraftUpdateRequest = {
  compositions: DraftCompositionSnapshot[];
};

export type PlayerStateSnapshot = {
  playerId: string;
  handCount: number;
  hand?: CardSnapshot[];
  totalPoints: number;
  pointsGained: number;
  unadjustedTotalPoints?: number;
  hasOpened: boolean;
  forfeited?: boolean;
};

export type TurnSnapshot = {
  number: number;
  playerIndex: number;
  playerId?: string;
  hasDrawn: boolean;
  mustUseDiscardDraw: boolean;
};

export type GameSnapshot = {
  gameMode?: GameMode;
  phase: number;
  round: number;
  dealerIndex: number;
  roundWinnerIndex: number;
  turn: TurnSnapshot;
  players: PlayerStateSnapshot[];
  hand: CardSnapshot[];
  drawPileCount: number;
  discardPile: CardSnapshot[];
  activeCompositions: CompositionSnapshot[];
  turnActivity?: TurnActivitySnapshot;
};

export function isFinishedGame(game: GameSnapshot) {
  // The backend's GamePhase enum is serialized as a number.
  return game.phase === 2 || game.phase === 3;
}

export type PlayerSnapshot = {
  playerId: string;
  userId?: string;
  sessionId?: string;
  name: string;
  imageUrl?: string;
  connected: boolean;
  seat: number;
  isHost: boolean;
  canReconnect: boolean;
  forfeited?: boolean;
  activeEmote?: {
    id: string;
    emoji: string;
    expiresAt: string;
  };
};

export type EndGameProposalSnapshot = {
  id: string;
  kind: "mutual_end" | "technical_abort";
  proposerPlayerId: string;
  description?: string;
  eligiblePlayerIds: string[];
  agreedPlayerIds: string[];
  expiresAt: string;
};

export type GameConclusionSnapshot = {
  kind: "forfeit" | "mutual_end" | "technical_abort";
  winnerPlayerId?: string;
  reportId?: string;
};

export type PendingDealChoiceSnapshot = {
  dealerIndex: number;
  chooserIndex: number;
  chooserPlayerId: string;
};

export type RoomSnapshot = {
  code: string;
  phase: string;
  gameMode?: GameMode;
  hostPlayerId: string;
  dealerIndex?: number;
  pendingDealChoice?: PendingDealChoiceSnapshot;
  endProposal?: EndGameProposalSnapshot;
  conclusion?: GameConclusionSnapshot;
  spectatorCount?: number;
  players: PlayerSnapshot[];
};

export type ActionResult = {
  requestId?: string;
  action: string;
  playerId: string;
  ok: boolean;
};

export type SocialUser = {
  id: string;
  name: string;
  imageUrl?: string;
  online: boolean;
  activeGame?: {
    startedAt: string;
  };
};

export type FriendRequest = {
  id: string;
  user: SocialUser;
  createdAt: string;
};

export type GameInvite = {
  id: string;
  user: SocialUser;
  roomCode: string;
  createdAt: string;
  expiresAt: string;
};

export type SocialState = {
  userId: string;
  friends: SocialUser[];
  incomingFriendRequests: FriendRequest[];
  outgoingFriendRequestUserIds: string[];
  gameInvites: GameInvite[];
};

type GameEventData = {
  connected: { sessionId: string; playerId: string };
  room_state: { room: RoomSnapshot };
  game_state: { room?: RoomSnapshot; game?: GameSnapshot | null; spectating?: boolean };
  action_result: ActionResult;
  social_state: Partial<SocialState>;
  error: { code?: string; message?: string; action?: string; requestId?: string };
  spectating_ended: never;
  left_room: never;
};

export type GameEvent = {
  [Type in keyof GameEventData]: { type: Type; data?: GameEventData[Type] };
}[keyof GameEventData];

const socialActions = [
  "send_friend_request",
  "respond_friend_request",
  "remove_friend",
  "send_game_invite",
  "respond_game_invite",
  "spectate_game",
  "stop_spectating",
] as const;

type SocialAction = (typeof socialActions)[number];

export type AcknowledgedAction =
  | SocialAction
  | "play"
  | "play_and_discard"
  | "discard"
  | "forfeit_game"
  | "request_end_game"
  | "vote_end_game"
  | "report_issue";

export function isSocialAction(action: string | undefined) {
  return socialActions.some((socialAction) => action === socialAction);
}

export function parseGameEvent(payload: unknown): GameEvent | null {
  if (typeof payload !== "string") return null;

  try {
    const event: { type?: unknown } | null = JSON.parse(payload);
    return typeof event?.type === "string" && event.type !== "" ? (event as GameEvent) : null;
  } catch {
    return null;
  }
}
