import {
  type CompletedGameSnapshot,
  type DealingChoiceRequest,
  type GameSnapshot,
  type GameMode,
  type PendingDealChoiceSnapshot,
  type PlayerSnapshot,
  type RoomSnapshot,
  type SocialState,
} from "#/components/game-websocket-provider";
import {
  AddCircleIcon,
  ChevronDownIcon,
  Copy01Icon,
  CopyLinkIcon,
  JokerIcon,
  Login02Icon,
  RankingIcon,
  Share08Icon,
  ZapIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { DealChoicePanel } from "#/components/game/deal-choice-panel";
import { PlayerEmotePicker } from "#/components/game/player-emotes";
import { PlayerStrip } from "#/components/game/player-strip";
import { AnimatedNumber } from "#/components/ui/animated-number";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "#/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "#/components/ui/dropdown-menu";
import { Input } from "#/components/ui/input";
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from "#/components/ui/field";
import { ToggleGroup, ToggleGroupItem } from "#/components/ui/toggle-group";
import { Caption, H2 } from "#/components/typography";
import { cn } from "#/lib/utils";
import { m } from "#/paraglide/messages.js";

type AsyncAction = () => Promise<void> | void;

type RoomActions = {
  canCreateRoom: boolean;
  canJoinRoom: boolean;
  canLeaveRoom: boolean;
  canStartGame: boolean;
  canSelectGameMode?: boolean;
};

type DealChoiceState = {
  pendingDealChoice: PendingDealChoiceSnapshot | null;
  dealChooserName: string | null;
  isDealChooser: boolean;
};

type GameLobbyViewProps = {
  room: RoomSnapshot | null;
  game: GameSnapshot | null;
  completedGame: CompletedGameSnapshot | null;
  players: PlayerSnapshot[];
  roomCode: string;
  gameMode?: GameMode;
  roomActions: RoomActions;
  dealChoice: DealChoiceState;
  onRoomCodeChange: (roomCode: string) => void;
  onCreateRoom: () => void;
  onJoinRoom: (roomCode: string) => void;
  onGameModeChange?: (gameMode: GameMode) => void;
  onStartGame: () => void;
  onChooseDealing: (choice: DealingChoiceRequest | string) => void;
  onLeaveRoom: () => void;
  onSendEmote: (emoji: string) => void;
  onCopyRoomCode: AsyncAction;
  onCopyRoomLink: AsyncAction;
  social?: SocialState;
  currentPlayerId?: string;
  onSendFriendRequest?: (userId: string) => Promise<unknown>;
};

export function GameLobbyView({
  room,
  game,
  completedGame,
  players,
  roomCode,
  gameMode = "full",
  roomActions,
  dealChoice,
  onRoomCodeChange,
  onGameModeChange,
  onCreateRoom,
  onJoinRoom,
  onStartGame,
  onChooseDealing,
  onLeaveRoom,
  onSendEmote,
  onCopyRoomCode,
  onCopyRoomLink,
  social,
  currentPlayerId,
  onSendFriendRequest,
}: GameLobbyViewProps) {
  const { canCreateRoom, canJoinRoom, canLeaveRoom, canStartGame, canSelectGameMode } = roomActions;
  const { pendingDealChoice, dealChooserName, isDealChooser } = dealChoice;
  const victorPlayerId = completedGame?.game.players[completedGame.game.roundWinnerIndex]?.playerId;
  const victor = players.find((player) => player.playerId === victorPlayerId) ?? null;
  const connectedCount = players.filter((player) => player.connected).length;
  const isHost = Boolean(room && currentPlayerId === room.hostPlayerId);

  return (
    <div
      className={cn(
        "mx-auto my-auto grid w-full gap-4 p-2",
        room ? "max-w-5xl lg:grid-cols-[minmax(0,1fr)_22rem]" : "max-w-lg",
      )}
    >
      <Card className="min-w-0 border border-border/70 shadow-sm">
        <CardHeader>
          <div
            className={cn(
              "flex flex-wrap justify-between gap-3",
              room ? "items-start" : "items-center",
            )}
          >
            <div className="flex min-w-0 items-center gap-3">
              {!room ? (
                <div className="flex size-10 shrink-0 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                  <HugeiconsIcon icon={JokerIcon} className="size-5" aria-hidden="true" />
                </div>
              ) : null}
              <div className="min-w-0">
                <CardTitle>{m.lobby()}</CardTitle>
                <CardDescription>{room ? m.ready_room() : m.create_or_join()}</CardDescription>
              </div>
            </div>
            {room ? (
              <Badge variant="secondary">
                <>
                  <AnimatedNumber value={connectedCount} />/
                  <AnimatedNumber value={players.length} /> {m.online()}
                </>
              </Badge>
            ) : null}
          </div>
        </CardHeader>
        <CardContent className="grid gap-4">
          {room ? (
            <div className="grid gap-4">
              <div className="grid gap-4 rounded-2xl border border-border/70 bg-muted/20 p-4">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <Caption className="font-medium tracking-[0.18em] uppercase">
                      {m.room_code()}
                    </Caption>
                    <H2 className="tabular-nums">{room.code}</H2>
                  </div>
                  <DropdownMenu>
                    <DropdownMenuTrigger render={<Button type="button" variant="outline" />}>
                      <HugeiconsIcon icon={Share08Icon} strokeWidth={2} data-icon="inline-start" />
                      {m.share()}
                      <HugeiconsIcon
                        icon={ChevronDownIcon}
                        strokeWidth={2}
                        data-icon="inline-end"
                      />
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end" className="w-48">
                      <DropdownMenuItem onClick={() => void onCopyRoomCode()}>
                        <HugeiconsIcon icon={Copy01Icon} strokeWidth={2} />
                        {m.copy_code()}
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => void onCopyRoomLink()}>
                        <HugeiconsIcon icon={CopyLinkIcon} strokeWidth={2} />
                        {m.copy_link()}
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              </div>
              {isHost ? (
                <FieldSet>
                  <FieldLegend variant="label">{m.game_mode()}</FieldLegend>
                  <ToggleGroup
                    value={[gameMode]}
                    onValueChange={(value) => {
                      const nextMode = value[0];
                      if (nextMode === "quick" || nextMode === "full") {
                        onGameModeChange?.(nextMode);
                      }
                    }}
                    disabled={!canSelectGameMode}
                    variant="outline"
                    spacing={0}
                    className="grid w-full grid-cols-2"
                    aria-label={m.game_mode()}
                  >
                    <ToggleGroupItem value="full">
                      <HugeiconsIcon icon={RankingIcon} data-icon="inline-start" />
                      {m.ranked()}
                    </ToggleGroupItem>
                    <ToggleGroupItem value="quick">
                      <HugeiconsIcon icon={ZapIcon} data-icon="inline-start" />
                      {m.quick()}
                    </ToggleGroupItem>
                  </ToggleGroup>
                  <FieldDescription>
                    {gameMode === "quick" ? m.quick_game_description() : m.full_game_description()}
                  </FieldDescription>
                </FieldSet>
              ) : null}
              <div className={cn("grid gap-2", isHost && "grid-cols-2")}>
                {isHost ? (
                  <Button type="button" onClick={onStartGame} disabled={!canStartGame}>
                    {m.start_game()}
                  </Button>
                ) : null}
                <Button
                  type="button"
                  variant="destructive"
                  onClick={onLeaveRoom}
                  disabled={!canLeaveRoom}
                >
                  {m.leave_room()}
                </Button>
              </div>
              {pendingDealChoice ? (
                <DealChoicePanel
                  players={players}
                  pendingDealChoice={pendingDealChoice}
                  dealChooserName={dealChooserName}
                  isDealChooser={isDealChooser}
                  onChooseDealing={onChooseDealing}
                />
              ) : null}
              {completedGame ? (
                <div className="grid gap-1 rounded-2xl border border-border/70 bg-muted/20 p-4">
                  <Caption className="font-medium tracking-[0.18em] uppercase">
                    {m.last_winner()}
                  </Caption>
                  <Caption>
                    {m.winner_in_round({
                      name: victor?.name ?? m.a_player(),
                      round: completedGame.game.round,
                    })}
                  </Caption>
                </div>
              ) : null}
            </div>
          ) : (
            <Button
              type="button"
              size="lg"
              className="w-full"
              onClick={onCreateRoom}
              disabled={!canCreateRoom}
            >
              <HugeiconsIcon icon={AddCircleIcon} data-icon="inline-start" />
              {m.create_room()}
            </Button>
          )}
          {!room ? (
            <>
              <CardDescription className="mx-auto">{m.or()}</CardDescription>
              <FieldGroup>
                <Field orientation="horizontal" className="items-stretch gap-2">
                  <FieldLabel htmlFor="room-code" className="sr-only">
                    {m.room_code()}
                  </FieldLabel>
                  <Input
                    id="room-code"
                    className="h-10"
                    value={roomCode}
                    onChange={(event) => onRoomCodeChange(event.target.value.toUpperCase())}
                    placeholder={m.room_code()}
                    maxLength={6}
                  />
                  <Button
                    type="button"
                    variant="outline"
                    size="lg"
                    onClick={() => onJoinRoom(roomCode.trim())}
                    disabled={!canJoinRoom}
                  >
                    <HugeiconsIcon icon={Login02Icon} data-icon="inline-start" />
                    {m.join()}
                  </Button>
                </Field>
              </FieldGroup>
            </>
          ) : null}
        </CardContent>
      </Card>

      {room ? (
        <div className="flex min-w-0 flex-col gap-4">
          <Card className="min-w-0 border border-border/70 shadow-sm">
            <CardHeader>
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <CardTitle>{m.players()}</CardTitle>
                  <CardDescription>{players.length ? m.seats() : m.no_room_yet()}</CardDescription>
                </div>
                {players.length ? (
                  <div className="flex items-center gap-2">
                    <PlayerEmotePicker onSendEmote={onSendEmote} />
                  </div>
                ) : null}
              </div>
            </CardHeader>
            <CardContent className="min-w-0">
              {players.length ? (
                <PlayerStrip
                  players={players}
                  game={game}
                  currentPlayerId={currentPlayerId}
                  social={social}
                  onSendFriendRequest={onSendFriendRequest}
                />
              ) : (
                <Caption className="rounded-2xl border border-dashed border-border/70 p-6">
                  {m.create_or_join_room()}
                </Caption>
              )}
            </CardContent>
          </Card>
        </div>
      ) : null}
    </div>
  );
}
