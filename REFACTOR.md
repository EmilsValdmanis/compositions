# Refactor decisions

## Game board

- Use one table-draft planner for validation, editing, the request, and spectator previews. The four paths separately grouped insertions, reversed entries at the start of a composition, and excluded joker replacements. Keep the start insertion order and separate reclaim operation required by the game.
- Give cards one value comparison, occurrence key, and face-down representation. Board state, composition rendering, dealing, and card animation used or defined those independently. Keep existing hand keys so saved order still applies.
- Let the board report when it finishes or skips the collection animation. The results route previously duplicated the board’s timings and could reveal results before the terminal game snapshot arrived. Schedule score reveals through one phase scheduler.
- Give the discard target one identity. The drag controller and drop zone used matching literals. Use the clipboard helper for copied room codes so the home page uses the same fallback as the other copy actions.
- Remove view props with no reader and tests that assert incidental object identity or historical animation constants. Keep the tests for visible order, validation, and animation behavior.

## Frontend server requests

- Share backend URL resolution and cookie forwarding at the frontend request boundary. Auth, admin, and leaderboard proxy calls each implemented the same forwarding policy. Keep public profile requests anonymous and keep private response cache policy where required.
- Read the request session directly and return the verification result at the game connection endpoint. The client only needed to know whether a session exists. Remove the unused session functions, the former auth library's call shape, and the Google-only sign-in method's provider switch.
- Define the sidebar state cookie next to its serialization policy. The router, shell, and sidebar each knew a piece of the cookie contract.
- Share the theme and motion bootstrap settings with the runtime. Keep the document’s motion attribute in sync when another tab changes or clears storage. Use one media query for the sidebar and toast placement.
- Calculate notification expiry from each incoming snapshot and schedule from the earliest live invite. An invite that arrived after expiring could otherwise remain visible indefinitely.
- Keep the development route guard and route identity together. The shared protected route and the development route both had to know the development URL exception.

## Realtime client

- Keep wire types and lobby state transitions outside the React provider. View code currently imports its data contract from a component that owns socket timers and lifecycle. Give each provider its own store, shared by its descendants.
- Use a single request identity for WebSocket commands and pending replies. Preserve reconnect backoff, resume after a transient failure, and pending action rejection on disconnect.
- Keep reply matching separate from state transitions. The server sends action results to multiple recipients, so promises match the initiating request and, for compatibility, its expected action and player.

## Backend server and storage

- Use one reply path for correlated successes and one for correlated failures. Game and social handlers routinely copy the request ID. The authentication failure path drops it. The forfeiting player still needs a direct reply because they leave the room's recipient list.
- Keep public errors attached to their codes or sentinels. The game had a duplicate string lookup alongside its sentinel lookup. Handle compatibility at the error boundary and preserve public messages.
- Make Postgres implement its interfaces directly where signatures already agree. Retain only the identity and lobby serialization adapters. Remove tests that merely run forwarding or unconfigured wrapper branches and use the coverage gates to verify the remaining executable behavior.
- Put the lobby's persistence, statistics queue, and turn activity in their respective files. Use the existing command lock as the single writer boundary, with readers still allowed during storage calls. The lobby command file currently also owns the snapshot layout, recovery validation, derived writes, and spectator activity. Preserve the stored format and the order of authoritative writes and statistics updates.
- Keep leaderboard metrics defined once for SQL scoring and render frontend tabs from the frontend contract. Use the storage layer’s issue length limit in the lobby.
- Share player state projection across regular updates and reconnect and use the same active-room lookup for social presence and spectating. Preserve private hands for players and masked hands for spectators.

## Game rules

- Put engine operations beside their rules. Round lifecycle and scoring, deal validation and execution, and the discard play search currently occupy the same file as turn commands. Preserve package boundaries and the public game API.
- Use the same game state copy rules for in-memory rollback and atomic play-and-discard. Remove standalone search and copy helpers used only by their own tests. Persisted snapshots validate historical records and have different needs from copying a live state.
- Plan both deal styles before mutating the round. An invalid next-round deal used to clear the existing state, and a dealt win could be scored under the prior round number.
- Give cards one identity comparison and share the opening points threshold between table play and discard search.
- Remove comments that narrate their adjacent implementation. Keep comments for compatibility, protocol behavior, and external constraints.

## Constraints that explain remaining special cases

- Historical lobby and game snapshots lack fields added later. Their readers continue to supply defaults.
- The terminal statistics queue survives room reset and restart. It serializes overlapping players' rating updates and can retry independently when the database is unavailable.
- WebSocket backoff, authentication revalidation and timeouts cover separate connection failure modes.
- A discarded card requires a legal play using that card. The engine retains its search and the frontend preserves the distinction between explicit and inferred joker reclaims.
- Reduced motion, missing animation geometry, duplicate cards, inactive seats, and onboarding each represent supported inputs or product behavior.
