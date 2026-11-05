package bonuses

import "github.com/google/uuid"

// One grant per account, so a sign-in told twice, or a second guest identity linked at once, earns one refill.
var signInGrants = uuid.MustParse("d8094223-2651-4709-a265-d23fe281da76")

func SignInGrantOf(holder Holder) GrantID {
	return GrantID(uuid.NewSHA1(signInGrants, []byte(holder)))
}
