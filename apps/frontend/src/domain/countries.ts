import countriesData from "../../static/countries/countries.json";

// The flags on the map, by code. Their names are the page's language's: translator().country(code).
export const Countries: ReadonlyMap<string, Country> = new Map(Object.keys(countriesData).map((code) => [code, {code}]));

export type Country = {
    code: string
}
