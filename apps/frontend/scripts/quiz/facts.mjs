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
 *   capitalIsWorldCity: boolean,
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

        if (facts.has(code)) continue

        const properties = feature.properties
        const capital = capitals.get(code)
        facts.set(code, {
            code,
            name,
            capital: capital?.name,
            capitalIsWorldCity: capital?.worldCity ?? false,
            continent: text(properties.CONTINENT),
            subregion: text(properties.SUBREGION),
            population: Number(properties.POP_EST) || 0,
            neighbours: [...(touching.get(code) ?? [])].filter((other) => names.has(other)).sort(),
        })
    }

    return facts
}

export function gameNames() {
    const file = path.join(frontendRoot, "static", "countries", "countries.json")
    return new Map(Object.entries(JSON.parse(fs.readFileSync(file, "utf8"))))
}

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
        // A country with two seats gets neither: the question needs one answer.
        if (capitals.has(code)) capitals.set(code, undefined)
        else if (name) capitals.set(code, {name, worldCity: properties.worldcity === 1})
    }

    return capitals
}

function text(value) {
    const string = String(value ?? "").trim().replace(/\s+/g, " ")
    return string && string !== "-99" ? string : undefined
}
