export type TableCompositionEdge = "start" | "end";

export const DISCARD_DROP_ID = "discard-pile";
export const HAND_DROP_ID = "hand-drop-zone";
export const NEW_COMPOSITION_DROP_ID = "new-composition-drop-zone";

const DRAFT_COMPOSITION_DROP_ID_PREFIX = "draft-composition-";
const TABLE_COMPOSITION_DROP_ID_PREFIX = "table-composition-";
const TABLE_COMPOSITION_EDGE_DROP_ID_PREFIX = "table-composition-edge-";
const TABLE_COMPOSITION_JOKER_DROP_ID_PREFIX = "table-composition-joker-";

export function draftCompositionDropId(compositionId: string) {
  return `${DRAFT_COMPOSITION_DROP_ID_PREFIX}${compositionId}`;
}

export function compositionIdFromDropId(dropId: string) {
  if (!dropId.startsWith(DRAFT_COMPOSITION_DROP_ID_PREFIX)) {
    return null;
  }

  return dropId.slice(DRAFT_COMPOSITION_DROP_ID_PREFIX.length);
}

export function tableCompositionDropId(compositionIndex: number) {
  return `${TABLE_COMPOSITION_DROP_ID_PREFIX}${compositionIndex}`;
}

export function tableCompositionEdgeDropId(compositionIndex: number, edge: TableCompositionEdge) {
  return `${TABLE_COMPOSITION_EDGE_DROP_ID_PREFIX}${compositionIndex}-${edge}`;
}

export function tableCompositionJokerDropId(compositionIndex: number, jokerIndex: number) {
  return `${TABLE_COMPOSITION_JOKER_DROP_ID_PREFIX}${compositionIndex}-${jokerIndex}`;
}

export function tableCompositionEdgeTargetFromDropId(dropId: string) {
  if (!dropId.startsWith(TABLE_COMPOSITION_EDGE_DROP_ID_PREFIX)) {
    return null;
  }

  const target = dropId.slice(TABLE_COMPOSITION_EDGE_DROP_ID_PREFIX.length);
  const separatorIndex = target.indexOf("-");
  if (separatorIndex < 0) {
    return null;
  }

  const compositionIndex = Number(target.slice(0, separatorIndex));
  const edge = target.slice(separatorIndex + 1);

  if (!Number.isInteger(compositionIndex) || compositionIndex < 0) {
    return null;
  }

  if (edge !== "start" && edge !== "end") {
    return null;
  }

  return {
    compositionIndex,
    edge,
  } as const;
}

export function tableCompositionJokerTargetFromDropId(dropId: string) {
  if (!dropId.startsWith(TABLE_COMPOSITION_JOKER_DROP_ID_PREFIX)) {
    return null;
  }

  const target = dropId.slice(TABLE_COMPOSITION_JOKER_DROP_ID_PREFIX.length);
  const separatorIndex = target.indexOf("-");
  if (separatorIndex < 0) {
    return null;
  }

  const compositionIndex = Number(target.slice(0, separatorIndex));
  const jokerIndex = Number(target.slice(separatorIndex + 1));

  if (!Number.isInteger(compositionIndex) || compositionIndex < 0) {
    return null;
  }

  if (!Number.isInteger(jokerIndex) || jokerIndex < 0) {
    return null;
  }

  return {
    compositionIndex,
    jokerIndex,
  } as const;
}
