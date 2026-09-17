-- Whether the sender was an admin when the message was sent. Messages from before are not.
ALTER TABLE messages ADD COLUMN author_admin boolean NOT NULL DEFAULT false;
