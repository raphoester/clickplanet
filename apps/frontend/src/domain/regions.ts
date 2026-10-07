import regions from "../../static/countries/regions.json"

const CONTINENTS: ReadonlyMap<string, string> = new Map(Object.entries(regions))

// The continent a country is in, as Natural Earth names it; written by `npm run regions`.
export function regionOf(country: string): string | undefined {
    return CONTINENTS.get(country)
}
