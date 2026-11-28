package name_flag_usecase

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles"
)

type Catalog interface {
	Name(ctx context.Context, code, name string, at time.Time) (nations.Flag, bool, error)
}

type Roles interface {
	Refresh(ctx context.Context, role roles.Role) error
}

// A sold flag's name, as the flags module told it: at its opening (and again at each boot), or the operator's rename.
type In struct {
	Code string
	Name string
	At   time.Time
}

func New(catalog Catalog, book Roles) *UseCase {
	return &UseCase{catalog: catalog, roles: book}
}

type UseCase struct {
	catalog Catalog
	roles   Roles
}

// The flag's role, when one was made, wears the name kept: asked after every fact, so a rename Discord refused once
// is tried again with the fact, and the restatement at each boot puts a name right. The role is made at the first
// grant, as a flag the game ships with's.
func (u *UseCase) Execute(ctx context.Context, in In) error {
	flag, _, err := u.catalog.Name(ctx, in.Code, in.Name, in.At)
	if err != nil {
		return err //nolint:wrapcheck // the catalog named it.
	}
	return u.roles.Refresh(ctx, roles.Nation(flag)) //nolint:wrapcheck // the book named it.
}
