package clicks

func ShieldError(owner, country string, shields, most int) error {
	switch {
	case country == "" || owner != country:
		return ErrNotYourTile
	case shields >= most:
		return ErrTileFull
	}

	return nil
}
