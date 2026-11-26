-- A flag the operator removed, for good: nobody pays for it, and its name is free for a new flag. Its own migration:
-- postgres uses a new value only once the transaction that added it is committed.
ALTER TYPE flag_state ADD VALUE 'removed';
