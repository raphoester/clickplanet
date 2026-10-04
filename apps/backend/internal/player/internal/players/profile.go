package players

import "time"

type Profile struct {
	account   AccountID
	name      Name
	updatedAt time.Time
	admin     bool
	color     Color
}

func NewProfile(account AccountID, name Name, at time.Time) Profile {
	return Profile{account: account, name: name, updatedAt: at}
}

func ProfileOf(account AccountID, name Name, updatedAt time.Time, admin bool, color Color) Profile {
	return Profile{account: account, name: name, updatedAt: updatedAt, admin: admin, color: color}
}

func (p Profile) Account() AccountID { return p.account }

func (p Profile) Name() Name { return p.name }

func (p Profile) UpdatedAt() time.Time { return p.updatedAt }

func (p Profile) Admin() bool { return p.admin }

func (p Profile) Color() Color { return p.color }
