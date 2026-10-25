# Manipulators are verbs, builders are nouns

**A name says whether calling it changes anything.** From Elegant Objects:

```
class Document {
    OutputPipe output();          // builder: a noun, reads, changes nothing
}

class OutputPipe {
    void write(InputStream stream); // manipulator: a verb, does the work
    int bytes();
    long time();
}
```

- **A manipulator performs an action and changes state**: it writes a row, sends a request, spends a token, mutates its receiver or an argument. It is named from a verb — `SaveSignIn`, `DeleteSessions`, `Session.Extend`, `Grant`, `Execute`.
- **A builder only reads or computes, and changes nothing** — not its receiver, not an argument, not the world. It is named from a noun, with `With`, `As` or `Of` where that reads better — `accounts.Cookie`, `accounts.ExpiredSessionCookie`, `store.Session(ctx, hash)`, `accounts.OutcomeOf`, `Flow.Challenge`, `Config.WithDefaults`, `oauth_http.Resource[T]`.
- **A verb on a pure function is a bug in the name.** `ClearCookie()` that only returns a `Set-Cookie` string reads as if it cleared something; it is `ExpiredSessionCookie()`, and the caller is what sends it. Likewise a read port is `Session(ctx, hash)`, not `FindSession`.
- **A builder that returns a boolean is an adjective** — `empty`, `readable`, `negative`: `Session.Linked()`, `Providers.Off()`, `Client.Configured()`, `Account.linkedTo(provider)`. Not `IsLinked`, `HasProvider` or `holds`. And the comma-ok second result of a builder (`CookieValue(…) (string, bool)`) is Go's idiom, not a name.
- **A manipulator returns no answer about what it did.** Ask first with an adjective, then act: `if session.Extendable(now, lifetime)`, then save `session.Extended(now, lifetime)`. Not `ExtendIfDue(...) bool`, which both acts and answers.
- **Prefer immutable values: a builder returns a changed copy, it does not change its receiver.** `session.Extended(now, lifetime)` is a new `*Session` with the new expiry, and `session` is left as it was; the only manipulator left is the store's `SaveSession`, which writes the copy. A method that sets a field on a domain value is the last resort, not the default.
- **A builder that returns an error is still a noun**: `Session.ExpiryError(now)`, `Flow.CallbackError(state, now)`, `Config.pruneError()`.
- **Two exceptions, both imposed from outside:** `Validate() error` is the hook `cpconfigs` calls by name, and a generated Connect handler method carries its RPC's name (`GetMe`).
- **Go's `New…` constructors stay** — building an object is what they say.

The auth module follows this. Code outside it has not been checked against the rule yet: fix a name when you touch the code, not in a sweep.
