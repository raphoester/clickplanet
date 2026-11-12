# Key Flow

```
POST /auth.v1.AuthService/CreateSession   [Cookie: cp_sid=…, sent by the web client with credentials]
  → [cpbootstrap: error net], RateLimitInterceptor (the mint budget)
  → AuthService → create_session_handler
  → create_session_usecase: attest (turnstile_attester → Cloudflare siteverify)
  → accounts: the cookie's live Session (extended and saved when due), or a GuestSession, stored
  → shared/cpsession.Signer.Mint [Ed25519 over version+expiry+id+account+linked+attested+scope; nothing stored]
  ← token, and Set-Cookie when the session is new or renewed

POST /auth.v1.AuthService/StartSignIn   [Cookie: cp_sid, read for a link only]
  → start_sign_in_usecase: a link needs the cookie's account; seal the flow with the intent
  ← authorization URL, Set-Cookie: cp_oauth (sealed flow)
  … the provider … → https://clickplanet.lol/auth/callback?code&state
POST /auth.v1.AuthService/CompleteSignIn   [Cookie: cp_oauth, cp_sid]
  → complete_sign_in_usecase: open and check the flow, a link's account, provider.Exchange, accounts.OutcomeOf, SaveSignIn
  ← Set-Cookie: cp_sid (new session), cp_oauth cleared; the client mints again

POST /auth.v1.AuthService/StartEmailSignIn   [Cookie: cp_sid, read for a link only]
  → start_email_sign_in_usecase: read the address, the blocklist, attest, a link's account, the address's budget, mail the code
  ← Set-Cookie: cp_email (sealed challenge)
POST /auth.v1.AuthService/CompleteEmailSignIn   [Cookie: cp_email, cp_sid]
  → complete_email_sign_in_usecase: open the challenge, spend a guess, check the code, then as CompleteSignIn
  ← Set-Cookie: cp_sid (new session), cp_email cleared; a wrong code keeps cp_email

POST /planet.v1.ClickService/Click   [X-Session-Token: <the minted token>]
  → [cpbootstrap: error net], CacheInterceptor, VPNBlockInterceptor,
      FreezeInterceptor (a frozen set refuses the play: FailedPrecondition, a FrozenRefusal), SessionInterceptor
      [props.Sessions: the key from auth.v1.InternalService, asked once per process
       then cpsession.Verifier: one signature check — this context holds no seed]
  → ClickService → click_handler
  → try_click       (tells the try, whatever came of it, and the bank it left: facts.Log, sent to Redis every 100ms; refuses nothing)
  → refusal_click   (a refused caller is PermissionDenied, with a BanRefusal: when the refusal ends)
  → allegiance_click (a flag other than the one pledged, or none pledged, is refused, and spends nothing)
  → spread_click    (with Spread: takes a spread click and the rings first, gives it back if the click is refused,
                     then claims the disc too; while the reach plays, the whole disc of a tile taken, else those in reach)
  → reach_click     (while the reach plays, a tile out of the flag's reach, or a spreading click whose disc is, is
                     FailedPrecondition with a ReachRefusal, and spends nothing)
  → throttle_click  (spends from the account's bucket and its scope's together, or refuses)
  → prom_click      (counts)
  → enclose_click   (with Enclose and an enclosure held: a take that closes a pocket fills it)
  → clicks/usecases/click_usecase (validates tile ID + country)
  → ledger.Recording.Click() [records a take or a strike]
  → clicks.Claiming.Click() → MemoryTileStorage.Claim [one write lock: read the tile, clicks.Tile.Claimed decides
      take, strike or nothing, write it, fortify after a take; each write fanned out in process as it lands]
  → every subscriber: one per open ListenForEvents stream

  … the try, as planet.v1.ClickTried on Redis
  → the offers' click_tried_subscriber (bonuses-clicks): a take, a strike or an unchanged tile marks the caller
      playing; a dry one owes a newcomer its first quiz (see bonuses.md)
  → the antibot's listener (antibot-clicks): Attempted, Inspect, Committed, as far as the click went
  → a guilty caller's ban row, and a planet.v1.RefuseCaller in the same transaction (the antibot's outbox)
  → planet's refuse_caller_handler (commands:planet.v1.RefuseCaller): planet.refused_callers, read by refusal_click
```

A claim's `Click` writes the tile as its `Set` does, and marks the update `clicked` when the owner changed: only the tile a click took carries it, and every client rings it. A strike, a spread's neighbours, an enclosure's inside and the moderation writes stay plain. A claim that changes nothing writes nothing and publishes nothing. Why the claim is one lock is in [A claim is one lock](tile-claim.md).

```
POST /chat.v1.ChatService/SendMessage   [X-Session-Token: required, naming an account]
  → [cpbootstrap: error net], BlocklistInterceptor, RateLimitInterceptor, then SessionInterceptor (a reader: refuses nothing)
      [props.Sessions: the key from auth.v1.InternalService, asked once per process]
  → ChatService → send_message_handler (the account off the context, or none)
  → muting_send_message: a mute on the account or its network is PermissionDenied, with when it ends
  → messages/usecases/send_message_usecase: no account is ErrNoAccount (Unauthenticated), then
      who posts (log_authors → rpc_player_authors → player.v1.InternalService/IntroducePlayer): a guest, which
      gets its code the first time and posts no GIF, or not; stamps id/time
      for a reply, the message it answers (messages.Thread: postgres_message_store.Message), by its account;
      a message that is not shown is ErrNotShown (NotFound)
  → postgres_message_store.Append() [inserts into chat.messages: the account, never a copy of its name;
      for a reply, the id of the message it answers]
  → inprocess_feed.Publish() → every subscriber: one per open ListenForEvents stream
  → publishing_send_message: chat.v1.MessageSent on the event bus, which player counts
```

A failed insert fails the whole post: the table is the audit trail, so a message nobody can account for later is not one that gets broadcast.

```
POST /chat.v1.ChatService/React   [X-Session-Token: required, naming an account]
  → [cpbootstrap: error net], BlocklistInterceptor, ReactionRateLimitInterceptor, SessionInterceptor (a reader)
  → react_handler (refuses a Reaction the proto does not name)
  → muting_react: a mute on the account or its network is PermissionDenied, with when it ends
  → reactions/usecases/react_usecase: who reacts is the account (reactions.ReactorOf); no account is ErrNoAccount
      is the message shown (postgres_message_store.Shown), what it carries (postgres_reaction_store.Reactions)
  → postgres_reaction_store.Save() [inserts or deletes in chat.reactions and bumps chat.reaction_versions, one statement]
      then reads the tally and its version back, one statement
  → inprocess_feed.Publish() the whole tally, versioned, by account: the page names who reacted
```
