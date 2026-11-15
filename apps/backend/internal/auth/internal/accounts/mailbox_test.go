package accounts_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

func TestGmailIgnoresTheDotsTheTagAndTheCase(t *testing.T) {
	for _, address := range []string{
		"johndoe@gmail.com", "John.Doe@Gmail.com", "j.o.h.n.d.o.e@gmail.com", "johndoe+clickplanet@gmail.com",
		"john.doe+a.b+c@gmail.com", " johndoe@GMAIL.COM ", "johndoe@googlemail.com", "john.doe+1@googlemail.com",
	} {
		assert.Equal(t, accounts.Mailbox("johndoe@gmail.com"), accounts.MailboxOf(address), address)
	}
}

func TestAPlusAddressedProviderIgnoresTheTagButNotTheDots(t *testing.T) {
	for address, mailbox := range map[string]string{
		"john.doe+news@outlook.com":   "john.doe@outlook.com",
		"John.Doe+x@Hotmail.fr":       "john.doe@hotmail.fr",
		"jane+game@icloud.com":        "jane@icloud.com",
		"jane+game@me.com":            "jane@me.com",
		"jane.doe+1@proton.me":        "jane.doe@proton.me",
		"jane+a@protonmail.com":       "jane@protonmail.com",
		"jane+a@fastmail.com":         "jane@fastmail.com",
		"ivan.petrov+game@yandex.ru":  "ivan.petrov@yandex.ru",
		"johndoe@outlook.com":         "johndoe@outlook.com",
		"john.doe@icloud.com":         "john.doe@icloud.com",
		"jane.doe+a+b@live.com":       "jane.doe@live.com",
		"jane.doe+@hotmail.com":       "jane.doe@hotmail.com",
		"me.com+tag@outlook.com":      "me.com@outlook.com",
		"someone+at+the@proton.me":    "someone@proton.me",
		"jane+game@yahoo.com":         "jane+game@yahoo.com",
		"jane.doe@googlemail.example": "jane.doe@googlemail.example",
	} {
		assert.Equal(t, accounts.Mailbox(mailbox), accounts.MailboxOf(address), address)
	}
}

func TestAnyOtherDomainIsItsOwnMailboxInLowerCase(t *testing.T) {
	for address, mailbox := range map[string]string{
		"John+Doe@Example.com":        "john+doe@example.com",
		"j.o.h.n@company.fr":          "j.o.h.n@company.fr",
		"jane+game@mail.gmail.com":    "jane+game@mail.gmail.com",
		"jane+game@gmail.com.example": "jane+game@gmail.com.example",
	} {
		assert.Equal(t, accounts.Mailbox(mailbox), accounts.MailboxOf(address), address)
	}
}

func TestANameTheRulesWouldEmptyIsKeptWhole(t *testing.T) {
	for address, mailbox := range map[string]string{
		"+tag@gmail.com": "+tag@gmail.com", "...@GMAIL.com": "...@gmail.com", "+@outlook.com": "+@outlook.com",
		"Nobody": "nobody", "": "",
	} {
		assert.Equal(t, accounts.Mailbox(mailbox), accounts.MailboxOf(address), address)
	}
}
