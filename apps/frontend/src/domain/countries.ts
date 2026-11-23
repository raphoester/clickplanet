import countriesData from "../../static/countries/countries.json";

// The flags the game ships with, by code: the server's list adds the ones opened since (flags.ts). Their names are the
// page's language's: translator().country(code).
export const Countries: ReadonlyMap<string, Country> = new Map(Object.keys(countriesData).map((code) => [code, {code}]));

export type Country = {
    code: string
}
