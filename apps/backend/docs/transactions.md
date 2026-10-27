# Transactions

**A use case opens a transaction and calls its stores and its outbox inside it as it does outside.** Nothing in the
use case names postgres, and no store is the one that holds the transaction:

```go
err = u.transactioner.InTx(ctx, func(ctx context.Context) error {
	if err := u.store.DeleteAccount(ctx, session.Account()); err != nil {
		return fmt.Errorf("failed to delete the account: %w", err)
	}
	if err := u.outbox.Record(ctx, &authv1.AccountDeleted{AccountId: session.Account().String()}); err != nil {
		return fmt.Errorf("failed to record the deletion: %w", err)
	}
	return nil
})
```

- **`shared/cpatomic`** holds the two ports a use case takes: `Transactioner` (`InTx(ctx, do)`) and `Outbox`
  (`Record(ctx, events...)`). `*cppg.Postgres` is the `Transactioner`; `cpoutbox.New(db)` is the `Outbox`. The module
  wires the same `db` into both and into the stores.
- **Recording an event is the use case's decision**, so the store does not take the event. `DeleteAccount`,
  `PruneGuests`, `SaveSignIn` and `rounds.Store.Close` write their rows and nothing else.

## How a store reaches postgres

**Only through `db.On(ctx)`.** A store holds `cppg.Conn`:

```go
type Conn interface {
	On(ctx context.Context) Querier
	InTx(ctx context.Context, do func(ctx context.Context) error) error
}
```

- **`On(ctx)` is the transaction `ctx` carries for this database, or the pool.** The store runs its SQL on that plain
  `Querier` (`s.db.On(ctx).ExecContext(ctx, ...)`). `Conn` has no `ExecContext`, and `*cppg.Postgres` is not a
  `Querier`, so a store cannot query without naming the context. Any other `Conn` must decide by the context too:
  the signature asks for one.
- **`InTx` joins the transaction `ctx` carries, or begins one.** A store's own multi-statement write
  (`s.db.InTx(ctx, func(ctx context.Context) error { tx := s.db.On(ctx); ... })`) is atomic alone, and part of the
  use case's transaction inside one. A failure or a panic rolls back. There is no `BeginTx` left outside `cppg`.
- **The transaction is keyed by the database.** A context that carries auth's transaction reaches scoring's pool
  as the pool.
- **`Outbox.Record` refuses outside a transaction** (`cppg.ErrNoTransaction`, through `InOpenTx`): an event is
  never kept apart from what it tells of.

## The one gap, and the rule that closes it

A write on the **outer** context inside `InTx` runs outside the transaction:
`TestAWriteOnTheOuterContextIsNotInTheTransaction` pins it. So the context `do` takes is always named `ctx`, which
hides the outer one. The ruleguard rule `transactionContext` (`tools/ruleguard/rules.go`) refuses any other name,
and revive's `unused-parameter` refuses a `ctx` that the closure never uses.

## Tests

- `cpoutbox/transaction_test.go` runs on a real postgres: a write and its event commit together, a failure keeps
  neither, a panic keeps nothing, an event outside a transaction is refused, one database's transaction never
  reaches another, the outer-context gap, and the fakes.
- A use case's test takes `cpatomic.FakeTransactioner{}` and a `&cpatomic.FakeOutbox{}` (behind the tag).
  `Recorded()` is what committed: the fake drops what a failed `do` recorded. An in-memory store keeps what a failed
  `do` wrote, so a rollback of the store is pinned in `cpoutbox`, once.
