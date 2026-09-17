package cpcountries

import "github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"

type Checker struct {
	countries *cpcolls.Set[string]
}

func New() *Checker {
	return &Checker{
		countries: countries,
	}
}

func (c *Checker) CheckCountry(s string) bool {
	return c.countries.Contains(s)
}
