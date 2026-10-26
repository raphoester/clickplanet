# Key Flow

```
POST /auth.v1.AuthService/CreateSession   [Cookie: cp_sid=…, sent by the web client with credentials]
  → [cpbootstrap: error net], RateLimitInterceptor (the mint budget)
  → AuthService → create_session_handler
  → create_session_usecase: attest (turnstile_attester → Cloudflare siteverify)
  → accounts: the cookie's live Session (extended and saved when due), or a GuestSession, stored
  → shared/cpsession.Signer.Mint [Ed25519 over version+expiry+id+account+scope; nothing stored]
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

POST /session.v1.SessionService/CreateSession   [deprecated]
  → create_anonymous_session_usecase: attest, then mint with no account

POST /planet.v1.ClickService/Click   [X-Session-Token: <the minted token>]
  → [cpbootstrap: error net], CacheInterceptor, VPNBlockInterceptor, SessionInterceptor
      [cpsessionverifier: the key from auth.v1.InternalService, asked once per boot
       then cpsession.Verifier: one signature check — this context holds no seed]
  → ClickService → click_handler
  → try_click       (tells the try, whatever came of it: facts.Log, sent to Redis every 100ms; refuses nothing)
  → refusal_click   (a refused caller is PermissionDenied, with a BanRefusal: when the refusal ends)
  → allegiance_click (a flag other than the one pledged, or none pledged, is refused, and spends nothing)
  → throttle_click  (spends from the account's bucket and its scope's together, or refuses)
  → prom_click      (counts)
  → clicks/usecases/click_usecase (validates tile ID + country)
  → ledger.Recording.Click() [records a take or a strike]
  → clicks.Claiming.Click() (asks clicks.Shielding: take, strike a shield, or nothing)
  → MemoryTileStorage.Click() [writes the flag; fans the update out in process, marked `clicked`]
  → every subscriber: one per open ListenForEvents stream

  … the try, as planet.v1.ClickTried on Redis
  → the antibot's listener (antibot-clicks): Attempted, Inspect, Committed, as far as the click went
  → a guilty caller's ban row, and a planet.v1.RefuseCaller in the same transaction (the antibot's outbox)
  → planet's refuse_caller_handler (commands:planet.v1.RefuseCaller): planet.refused_callers, read by refusal_click
```

`Click` is `Set` with the update marked `clicked`, which only the tile a click named carries: every client rings it, and a spread's neighbours, an enclosure's inside and the moderation writes stay plain `Set`s. Both are a no-op when the tile already holds that value — no write, no update published.

```
POST /chat.v1.ChatService/SendMessage   [X-Session-Token: required, naming an account]
  → [cpbootstrap: error net], BlocklistInterceptor, RateLimitInterceptor, then SessionInterceptor (a reader: refuses nothing)
      [cpsessionverifier: the key from auth.v1.InternalService, asked once per boot]
  → ChatService → send_message_handler (the account off the context, or none)
  → muting_send_message: a mute on the account or its network is PermissionDenied, with when it ends
  → messages/usecases/send_message_usecase: no account is ErrNoAccount (Unauthenticated), then
      who posts (log_authors → rpc_player_authors → player.v1.InternalService/GetAuthor): the account's
      username, or "guest_" and its guest code; stamps id/time
      for a reply, the message it answers (messages.Thread: postgres_message_store.Message, then GetAuthors);
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
  → who everyone under the message is, one ask (rpc_player_authors.Authors), for the answer and the frame alike
  → inprocess_feed.Publish() the whole tally, versioned and named
```
