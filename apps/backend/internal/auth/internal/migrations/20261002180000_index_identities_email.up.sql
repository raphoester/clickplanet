CREATE INDEX identities_email ON identities (lower(email)) WHERE email_verified;
