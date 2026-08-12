export const ROOM_CODE_LENGTH = 6;

export function isCompleteRoomCode(roomCode: string) {
  return roomCode.trim().length === ROOM_CODE_LENGTH;
}
