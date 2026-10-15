package players

import "time"

type Profile struct {
	account   AccountID
	name      Name
	updatedAt time.Time
	choice    NameChoice
	admin     bool
	color     Color
}

func NewProfile(account AccountID, name Name, at time.Time) Profile {
	return ProfileSnapshot{Account: account, Name: name, UpdatedAt: at}.Restore()
}

func Unnamed(account AccountID) Profile {
	return ProfileSnapshot{Account: account}.Restore()
}

func (p Profile) Renamed(name Name, at time.Time) (Profile, error) {
	if err := p.choice.RenameError(at); err != nil {
		return Profile{}, err
	}

	p.name = name
	p.updatedAt = at
	p.choice = NameChoice{at: at}
	return p, nil
}

func (p Profile) TakeSnapshot() ProfileSnapshot {
	return ProfileSnapshot{
		Account: p.account, Name: p.name, UpdatedAt: p.updatedAt, ChosenAt: p.choice.at, Admin: p.admin, Color: p.color,
	}
}

type ProfileSnapshot struct {
	Account   AccountID
	Name      Name
	UpdatedAt time.Time
	ChosenAt  time.Time
	Admin     bool
	Color     Color
}

func (s ProfileSnapshot) Restore() Profile {
	return Profile{
		account: s.Account, name: s.Name, updatedAt: s.UpdatedAt, choice: NameChosenAt(s.ChosenAt), admin: s.Admin,
		color: s.Color,
	}
}
