// What the generator knows about each country, from the two sources that already decide the map.
//
// The *name* is the game's own (`static/countries/countries.json`), never Natural Earth's. A
// question is read beside a flag and a leaderboard row, and those say "Samoa, USA" where Natural
// Earth says "American Samoa" — a quiz that calls a country something the rest of the screen does
// not is a quiz that looks wrong even when it is right. A country the game has no name for gets no
// questions at all: it cannot be a right answer the player could pick out, and it cannot be a
// plausible wrong one either.
import fs from "node:fs"
import path from "node:path"
import {fileURLToPath} from "node:url"

import {countryCodeOf} from "../map/ground.mjs"
import {naturalEarth} from "../map/naturalEarth.mjs"
import {adjacency} from "./adjacency.mjs"

const frontendRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..", "..")

/**
 * @typedef {{
 *   code: string,
 *   name: string,
 *   capital: string | undefined,
 *   continent: string | undefined,
 *   subregion: string | undefined,
 *   population: number,
 *   neighbours: string[],
 * }} Facts
 */

/** @returns {Promise<Map<string, Facts>>} keyed by the country code the map uses */
export async function countryFacts({directory} = {}) {
    const names = gameNames()
    const [countries, places] = await Promise.all([
        naturalEarth("ne_50m_admin_0_countries"),
        naturalEarth("ne_50m_populated_places_simple"),
    ])

    const capitals = capitalsByCountry(places, countries)
    const touching = adjacency(directory)

    const facts = new Map()
    for (const feature of countries.features) {
        const code = countryCodeOf(feature)
        const name = code && names.get(code)
        if (!name) continue

        // Natural Earth maps a few countries as more than one feature; the first carries the
        // sovereign's own row, and a second would only overwrite it with the same answers.
        if (facts.has(code)) continue

        const properties = feature.properties
        facts.set(code, {
            code,
            name,
            capital: capitals.get(code),
            continent: text(properties.CONTINENT),
            subregion: text(properties.SUBREGION),
            population: Number(properties.POP_EST) || 0,
            // Sorted, so two runs of the generator produce the same bytes.
            neighbours: [...(touching.get(code) ?? [])].filter((other) => names.has(other)).sort(),
        })
    }

    return facts
}

/** The names the game shows, which are the only names a question may use. */
export function gameNames() {
    const file = path.join(frontendRoot, "static", "countries", "countries.json")
    return new Map(Object.entries(JSON.parse(fs.readFileSync(file, "utf8"))))
}

// `adm0cap` is Natural Earth's own mark for the seat of a sovereign state, which is what "capital"
// means in a question. `iso_a2` on a place is occasionally missing where the country file has one,
// so the three-letter admin code is the second way in.
function capitalsByCountry(places, countries) {
    const codeOfA3 = new Map()
    for (const feature of countries.features) {
        const code = countryCodeOf(feature)
        if (code) codeOfA3.set(String(feature.properties.ADM0_A3), code)
    }

    const capitals = new Map()
    for (const feature of places.features) {
        const properties = feature.properties
        if (properties.adm0cap !== 1) continue

        const iso = String(properties.iso_a2 ?? "").toLowerCase()
        const code = iso && iso !== "-99" ? iso : codeOfA3.get(String(properties.adm0_a3))
        if (!code) continue

        const name = text(properties.name)
        // A country with two seats gets neither: "the capital" has to have one answer.
        if (capitals.has(code)) capitals.set(code, undefined)
        else if (name) capitals.set(code, name)
    }

    return capitals
}

// Natural Earth's own spacing is not always one space ("Washington,  D.C."), and the string is
// going on screen as a choice to press.
function text(value) {
    const string = String(value ?? "").trim().replace(/\s+/g, " ")
    return string && string !== "-99" ? string : undefined
}
