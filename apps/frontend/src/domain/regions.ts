import regions from "../../static/countries/regions.json"

const CONTINENTS: ReadonlyMap<string, string> = new Map(Object.entries(regions))

// The continent a country is in, as Natural Earth names it, or in Asia, too big to be one place, the part of it: the
// Middle East, South Asia, East Asia, Southeast Asia, Central Asia. Written by `npm run regions`.
export function regionOf(country: string): string | undefined {
    return CONTINENTS.get(country)
}

// The only continent with a flag of its own in static/countries/svg.
const CONTINENT_FLAGS: ReadonlyMap<string, string> = new Map([["Europe", "eu"]])

export function flagOfContinent(continent: string): string | undefined {
    return CONTINENT_FLAGS.get(continent)
}
