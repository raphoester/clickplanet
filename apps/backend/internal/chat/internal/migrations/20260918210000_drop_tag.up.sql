-- The tag was a salted hash of the sender's address, shown beside its name. A guest is now named by its guest
-- code, and the address is never public, so the column goes. The address itself stays in ip, for moderation.
ALTER TABLE messages DROP COLUMN tag;
