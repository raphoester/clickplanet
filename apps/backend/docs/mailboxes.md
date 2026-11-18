# One mailbox, one player (`accounts.MailboxOf`)

**Every form of an address that reaches one inbox is one identity.** Until 2026-11-15 the identity of an emailed code
was the address in lower case, so `johndoe@gmail.com`, `john.doe@gmail.com`, `johndoe+1@gmail.com` and
`johndoe@googlemail.com` were four accounts out of one inbox, each of them signed in: the linked refill, never pruned,
a sign-in refill each ([A guest's first sign-in](sign-in-refill.md)). The disposable domains were refused, and this
was the free way around them.

## The rule

- **`accounts.MailboxOf(address)`** is the address trimmed and in lower case, with what its provider ignores cut: for
  the providers of `plusAddressed`, the `+` and what follows it in the name; for Gmail the dots too, and
  `googlemail.com` is `gmail.com`. Every other address is its own mailbox. A name the rules would empty (`+x@gmail.com`)
  is kept whole.
- **Gmail alone ignores the dots.** Outlook's, iCloud's, Proton's and the others' dots are part of the name.
- **A list, not every domain.** Elsewhere a `+` may be part of a name (Postfix ships with no `recipient_delimiter`),
  and a mailbox that matches signs in: a wrong fold would sign a stranger in to another player's account. A custom
  domain is its own mailbox.
- **Why the listed ones are safe to fold, even where the provider says little** (Outlook and iCloud document plus
  addressing thinly; Gmail, Proton and Fastmail document it): none of them allows a `+` in a name of its own. So an
  address with one is either an alias of the name before it, or reaches nobody and no code arrives.
- **The list**: Gmail (`gmail.com`, `googlemail.com`); Microsoft (`outlook.com`, `.fr`, `.de`, `.es`, `.it`,
  `hotmail.com`, `.fr`, `.co.uk`, `.de`, `.it`, `.es`, `live.com`, `.fr`, `.co.uk`, `msn.com`); Apple (`icloud.com`,
  `me.com`, `mac.com`); Proton (`proton.me`, `protonmail.com`, `protonmail.ch`, `pm.me`); Fastmail (`fastmail.com`,
  `fastmail.fm`); Yandex (`yandex.ru`, `yandex.com`, `ya.ru`). `mailbox_test.go` pins each kind.
- **A domain added later changes the mailbox of addresses already kept**: it comes with a migration that folds them
  again, as `20261115123255_identity_mailbox` did. Without one, an identity kept before keeps its old mailbox and a
  new form of its address makes a new account.

## Where it is used

- **An emailed code's identity**: its subject is the mailbox, and `email` the address as typed (trimmed, in lower
  case), where every code goes, so a player's filter on `jane+game@gmail.com` still catches them
  (`signin.Challenge.Claim`).
- **Every verified address** (Google's and Discord's too) keeps its mailbox in `identities.mailbox`, which
  `Identity.Mailbox` derives from the address. **`Store.AccountOfMailbox`** finds its owner: the account of the oldest
  identity holding it (index `identities_mailbox`). So a form of an address a Google or Discord account verified joins
  that account (`Joined`), and a link of it from another account is refused (`ErrIdentityLinkedElsewhere`), as for
  the address itself ([Auth](auth.md)).
- **The code budget is per mailbox** (`signin.Post`): three forms of one address are one inbox to fill.

## The migration `20261115123255_identity_mailbox`

- **It folds the rows kept with the same rules in SQL**, once: a `pg_temp` function, gone when the migration ends, so
  Go stays the one copy from then on.
- **The owner of each mailbox keeps one code**: the account of the mailbox's oldest verified identity, its oldest
  emailed code's identity, whose subject becomes the mailbox. **Every other emailed code's identity of that mailbox
  gives it up**: its subject becomes `alias:<account>:<address>`, which no sign-in asks for, and its address stays, so
  the operator's `player.v1.AdminService/SearchPlayersByEmail` still finds it ([Player](player.md)). Its account keeps
  its sessions and its other identities. The next code to any form of the address signs in to the owner, or joins it
  when the owner holds no code yet.
- **It bans nobody.** The accounts the forms of an address made are the antibot's to judge (below).
- **The down gives each code its address back**, the keepers first. It fails when a sign-in after the up linked a form
  of an address an identity given up holds: two identities of one address, as before.

## The accounts the forms of an address made

**The operator decided on 2026-11-15: each of them is banned for good, and the owner is warned.** It is done by hand,
with the operator's tools: `SearchPlayersByEmail` (`+`, `@gmail.com`, `@googlemail.com`) for the accounts, grouped by
mailbox; `antibot.v1.AdminService/BanPlayer` with `account_id` and a duration of 100 years (`3153600000s`) for each
account that is not its mailbox's owner; `WarnPlayer` for the owner, once ([Operator tools](admin.md)). Two forms used
by one player apart on purpose (a `+discord` address for Discord) read the same, so the list is read by a person
before the bans.

## What it cannot see

A domain not in the list (a custom domain, its catch-all), the relay services that hand out addresses at random
(DuckDuckGo's `duck.com`, Apple's Hide My Email, Firefox Relay, SimpleLogin, addy.io: none is in the disposable list),
the aliases of other names that Outlook gives one account, and a second mailbox. Those are the antibot's: playing
several accounts in turns is the rotator's ([Anti-bot](antibot.md)).
