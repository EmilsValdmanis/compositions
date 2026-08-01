import {
  type GameSnapshot,
  type PlayerSnapshot,
  type RoomSnapshot,
} from "#/components/game-websocket-provider";
import { m } from "#/paraglide/messages.js";

export type MockScenario = {
  id: string;
  label: string;
  description: string;
  room: RoomSnapshot;
  game: GameSnapshot;
  players: PlayerSnapshot[];
  controlledPlayerId: string;
};

export const mockScenarios: MockScenario[] = [
  {
    id: "table-activity-showcase",
    label: m.mock_table_activity(),
    description: m.mock_table_activity_description(),
    controlledPlayerId: "player-avery",
    players: [
      {
        playerId: "player-avery",
        sessionId: "session-avery",
        name: "Avery",
        connected: true,
        seat: 0,
        isHost: true,
        canReconnect: true,
      },
      {
        playerId: "player-blair",
        sessionId: "session-blair",
        name: "Blair",
        connected: true,
        seat: 1,
        isHost: false,
        canReconnect: true,
      },
      {
        playerId: "player-casey",
        sessionId: "session-casey",
        name: "Casey",
        connected: true,
        seat: 2,
        isHost: false,
        canReconnect: true,
      },
      {
        playerId: "player-devon",
        sessionId: "session-devon",
        name: "Devon",
        connected: false,
        seat: 3,
        isHost: false,
        canReconnect: true,
      },
    ],
    room: {
      code: "DEV01",
      phase: "in_progress",
      hostPlayerId: "player-avery",
      players: [
        {
          playerId: "player-avery",
          sessionId: "session-avery",
          name: "Avery",
          connected: true,
          seat: 0,
          isHost: true,
          canReconnect: true,
        },
        {
          playerId: "player-blair",
          sessionId: "session-blair",
          name: "Blair",
          connected: true,
          seat: 1,
          isHost: false,
          canReconnect: true,
        },
        {
          playerId: "player-casey",
          sessionId: "session-casey",
          name: "Casey",
          connected: true,
          seat: 2,
          isHost: false,
          canReconnect: true,
        },
        {
          playerId: "player-devon",
          sessionId: "session-devon",
          name: "Devon",
          connected: false,
          seat: 3,
          isHost: false,
          canReconnect: true,
        },
      ],
    },
    game: {
      phase: 1,
      round: 4,
      dealerIndex: 3,
      roundWinnerIndex: 1,
      turn: {
        number: 9,
        playerIndex: 0,
        playerId: "player-avery",
        hasDrawn: true,
        mustUseDiscardDraw: false,
      },
      players: [
        {
          playerId: "player-avery",
          handCount: 14,
          totalPoints: 20,
          pointsGained: 20,
          hasOpened: true,
        },
        {
          playerId: "player-blair",
          handCount: 6,
          totalPoints: 30,
          pointsGained: 0,
          hasOpened: true,
        },
        {
          playerId: "player-casey",
          handCount: 9,
          totalPoints: 89,
          unadjustedTotalPoints: 107,
          pointsGained: 12,
          hasOpened: false,
        },
        {
          playerId: "player-devon",
          handCount: 11,
          totalPoints: 42,
          pointsGained: 42,
          hasOpened: true,
        },
      ],
      hand: [
        { rank: 4, suit: 1 },
        { rank: 5, suit: 1 },
        { rank: 6, suit: 1 },
        { rank: 9, suit: 2 },
        { rank: 9, suit: 3 },
        { rank: 10, suit: 2 },
        { rank: 12, suit: 1 },
        { rank: 11, suit: 2 },
        { rank: 12, suit: 2 },
        { isJoker: true },
        { rank: 1, suit: 0 },
        { rank: 13, suit: 3 },
        { rank: 3, suit: 2 },
        { rank: 7, suit: 1 },
      ],
      drawPileCount: 27,
      discardPile: [
        { rank: 7, suit: 3 },
        { rank: 2, suit: 0 },
        { rank: 13, suit: 1 },
      ],
      activeCompositions: [
        {
          type: "run",
          cards: [
            { rank: 7, suit: 1 },
            { rank: 8, suit: 1 },
            { rank: 9, suit: 1 },
            { rank: 10, suit: 1 },
          ],
          points: 40,
          complete: false,
        },
        {
          type: "set",
          cards: [
            { rank: 12, suit: 0 },
            { isJoker: true },
            { rank: 12, suit: 3 },
            { rank: 12, suit: 2 },
          ],
          jokerRepresentations: {
            1: [{ rank: 12, suit: 1 }],
          },
          points: 55,
          complete: false,
        },
        {
          type: "run",
          cards: [
            { rank: 1, suit: 2 },
            { rank: 2, suit: 2 },
            { rank: 3, suit: 2 },
            { rank: 4, suit: 2 },
          ],
          points: 30,
          complete: false,
        },
        {
          type: "set",
          cards: [
            { rank: 5, suit: 0 },
            { rank: 5, suit: 2 },
            { rank: 5, suit: 3 },
          ],
          points: 20,
          complete: false,
        },
        {
          type: "run",
          cards: [
            { rank: 2, suit: 3 },
            { rank: 3, suit: 3 },
            { rank: 4, suit: 3 },
            { rank: 5, suit: 3 },
            { rank: 6, suit: 3 },
            { rank: 7, suit: 3 },
            { rank: 8, suit: 3 },
            { rank: 9, suit: 3 },
            { rank: 10, suit: 3 },
          ],
          points: 54,
          complete: false,
        },
        {
          type: "set",
          cards: [
            { rank: 13, suit: 0 },
            { rank: 13, suit: 1 },
            { rank: 13, suit: 2 },
            { rank: 13, suit: 3 },
          ],
          points: 40,
          complete: true,
        },
        {
          type: "run",
          cards: [
            { rank: 9, suit: 2 },
            { rank: 10, suit: 2 },
            { rank: 11, suit: 2 },
            { rank: 12, suit: 2 },
            { rank: 13, suit: 2 },
          ],
          points: 50,
          complete: false,
        },
      ],
      turnActivity: {
        playerId: "player-avery",
        round: 4,
        turnNumber: 9,
        baselineCompositions: [
          {
            type: "run",
            cards: [
              { rank: 7, suit: 1 },
              { rank: 8, suit: 1 },
              { rank: 9, suit: 1 },
              { rank: 10, suit: 1 },
            ],
            points: 40,
            complete: false,
          },
          {
            type: "set",
            cards: [{ rank: 12, suit: 0 }, { isJoker: true }, { rank: 12, suit: 3 }],
            jokerRepresentations: {
              1: [{ rank: 12, suit: 1 }],
            },
            points: 45,
            complete: false,
          },
          {
            type: "run",
            cards: [
              { rank: 1, suit: 2 },
              { rank: 2, suit: 2 },
              { rank: 3, suit: 2 },
              { rank: 4, suit: 2 },
            ],
            points: 30,
            complete: false,
          },
        ],
        draftCompositions: [
          {
            id: "draft-new-avery",
            cards: [
              { rank: 4, suit: 1 },
              { rank: 5, suit: 1 },
              { rank: 6, suit: 1 },
            ],
          },
          {
            id: "draft-table-2-avery",
            tableIndex: 2,
            cards: [{ rank: 5, suit: 2 }],
          },
        ],
        compositionActivities: [
          {
            tableIndex: 3,
            kind: "new_composition",
            playerId: "player-avery",
            cardActivities: {
              0: {
                kind: "new_composition",
                playerId: "player-avery",
              },
              1: {
                kind: "new_composition",
                playerId: "player-avery",
              },
              2: {
                kind: "new_composition",
                playerId: "player-avery",
              },
            },
          },
        ],
      },
    },
  },
  {
    id: "other-player-turn",
    label: m.mock_other_player_turn(),
    description: m.mock_other_player_turn_description(),
    controlledPlayerId: "player-avery",
    players: [
      {
        playerId: "player-avery",
        sessionId: "session-avery",
        name: "Avery",
        connected: true,
        seat: 0,
        isHost: true,
        canReconnect: true,
      },
      {
        playerId: "player-blair",
        sessionId: "session-blair",
        name: "Blair",
        connected: true,
        seat: 1,
        isHost: false,
        canReconnect: true,
      },
      {
        playerId: "player-casey",
        sessionId: "session-casey",
        name: "Casey",
        connected: true,
        seat: 2,
        isHost: false,
        canReconnect: true,
      },
      {
        playerId: "player-devon",
        sessionId: "session-devon",
        name: "Devon",
        connected: true,
        seat: 3,
        isHost: false,
        canReconnect: true,
      },
    ],
    room: {
      code: "DEV02",
      phase: "in_progress",
      hostPlayerId: "player-avery",
      players: [
        {
          playerId: "player-avery",
          sessionId: "session-avery",
          name: "Avery",
          connected: true,
          seat: 0,
          isHost: true,
          canReconnect: true,
        },
        {
          playerId: "player-blair",
          sessionId: "session-blair",
          name: "Blair",
          connected: true,
          seat: 1,
          isHost: false,
          canReconnect: true,
        },
        {
          playerId: "player-casey",
          sessionId: "session-casey",
          name: "Casey",
          connected: true,
          seat: 2,
          isHost: false,
          canReconnect: true,
        },
        {
          playerId: "player-devon",
          sessionId: "session-devon",
          name: "Devon",
          connected: true,
          seat: 3,
          isHost: false,
          canReconnect: true,
        },
      ],
    },
    game: {
      phase: 1,
      round: 6,
      dealerIndex: 0,
      roundWinnerIndex: 2,
      turn: {
        number: 14,
        playerIndex: 2,
        playerId: "player-casey",
        hasDrawn: true,
        mustUseDiscardDraw: false,
      },
      players: [
        {
          playerId: "player-avery",
          handCount: 10,
          totalPoints: 36,
          pointsGained: 18,
          hasOpened: true,
        },
        {
          playerId: "player-blair",
          handCount: 7,
          totalPoints: 18,
          pointsGained: 18,
          hasOpened: true,
        },
        {
          playerId: "player-casey",
          handCount: 6,
          totalPoints: 8,
          pointsGained: 0,
          hasOpened: true,
        },
        {
          playerId: "player-devon",
          handCount: 12,
          totalPoints: 52,
          pointsGained: 52,
          hasOpened: false,
        },
      ],
      hand: [
        { rank: 2, suit: 1 },
        { rank: 3, suit: 1 },
        { rank: 4, suit: 1 },
        { rank: 8, suit: 0 },
        { rank: 8, suit: 1 },
        { rank: 8, suit: 2 },
      ],
      drawPileCount: 18,
      discardPile: [
        { rank: 6, suit: 3 },
        { rank: 11, suit: 0 },
      ],
      activeCompositions: [
        {
          type: "run",
          cards: [
            { rank: 9, suit: 0 },
            { rank: 10, suit: 0 },
            { rank: 11, suit: 0 },
            { rank: 12, suit: 0 },
          ],
          points: 40,
          complete: false,
        },
        {
          type: "set",
          cards: [
            { rank: 8, suit: 0 },
            { rank: 8, suit: 1 },
            { rank: 8, suit: 3 },
          ],
          points: 30,
          complete: false,
        },
      ],
      turnActivity: {
        playerId: "player-casey",
        round: 6,
        turnNumber: 14,
        baselineCompositions: [
          {
            type: "run",
            cards: [
              { rank: 9, suit: 0 },
              { rank: 10, suit: 0 },
              { rank: 11, suit: 0 },
              { rank: 12, suit: 0 },
            ],
            points: 40,
            complete: false,
          },
          {
            type: "set",
            cards: [
              { rank: 8, suit: 0 },
              { rank: 8, suit: 1 },
              { rank: 8, suit: 3 },
            ],
            points: 30,
            complete: false,
          },
        ],
        draftCompositions: [
          {
            id: "draft-new-casey",
            cards: [
              { rank: 2, suit: 1 },
              { rank: 3, suit: 1 },
              { rank: 4, suit: 1 },
            ],
          },
          {
            id: "draft-table-0-casey",
            tableIndex: 0,
            cards: [{ rank: 13, suit: 0 }],
          },
        ],
        compositionActivities: [
          {
            tableIndex: 0,
            cardActivities: {
              4: {
                kind: "addition",
                playerId: "player-casey",
              },
            },
          },
        ],
      },
    },
  },
];
