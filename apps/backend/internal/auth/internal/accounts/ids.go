package accounts

import "github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"

// AccountID names an account. It is the click token's own type, so an account reaches the token with no conversion.
type AccountID = cpsession.AccountID

// TokenHash names a session: the SHA-256 of its cookie's token, the only form of it that is stored.
type TokenHash []byte
