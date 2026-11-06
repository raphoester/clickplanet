CREATE TYPE moderation_action AS ENUM ('mute', 'unmute', 'removal');

-- What an operator did to an account, kept past the chat's retention, for chat.storage.moderationRetention.
CREATE TABLE moderation (
    id          bigserial         PRIMARY KEY,
    -- The account it was taken on, never the network a mute also holds. NULL for a message kept before accounts.
    account_id  uuid,
    action      moderation_action NOT NULL,
    acted_at    timestamptz       NOT NULL,
    -- A mute's and an unmute's.
    mute_kind   mute_kind,
    -- A mute's.
    seconds     integer,
    -- A removal's: the message, and a copy of what it said, with no address.
    message_id  text,
    text        text,
    gif_url     text,
    gif_width   integer,
    gif_height  integer,
    reason      text              NOT NULL,
    -- NULL for a call on the console listener, and once the operator's account is forgotten.
    operator_id uuid,
    client      text              NOT NULL,
    CHECK ((action = 'removal') = (message_id IS NOT NULL AND text IS NOT NULL)),
    CHECK ((action = 'removal') = (mute_kind IS NULL)),
    CHECK ((action = 'mute') = (seconds IS NOT NULL)),
    CHECK ((gif_url IS NULL) = (gif_width IS NULL) AND (gif_url IS NULL) = (gif_height IS NULL))
);

-- A message is removed once: a call tried again never counts twice.
CREATE UNIQUE INDEX moderation_removal_key ON moderation (message_id) WHERE action = 'removal';

CREATE INDEX moderation_account_idx ON moderation (account_id, acted_at DESC);

CREATE INDEX moderation_acted_at_idx ON moderation (acted_at DESC);
