import {keyOfName} from "./labels.mjs"

// A geographic name beats an administrative one that fits as well: "Vanua Levu", not "Northern Division".
// A shared layer is not cut by country, so Margarita is not "Lesser Antilles" for being Venezuela's only tile there.
const LAYERS = [
    {id: "subunit50", weight: 1},
    {id: "subunit10", weight: 1},
    {id: "island", weight: 1, shared: true},
    {id: "islandGroup", weight: 0.9, shared: true},
    {id: "islandPoint", weight: 0.85, shared: true},
    {id: "admin1", weight: 0.8},
    {id: "admin1Region", weight: 0.75},
    {id: "landform", weight: 0.6, shared: true},
]

// A landmass across two provinces is named by neither.
const MAJORITY = 0.5
// Below this a feature is where the landmass is ("Scotland"), not its name.
const FLOOR = 0.1
const CAPITAL_FIT = 0.7

const cleaned = (name) => String(name).replace(/\s*\([^)]*\)\s*/g, " ").replace(/\s+/g, " ").trim()

/**
 * One name per landmass, unique within its country. The country's main landmass carries the country's name.
 * @param {ReturnType<import("./evidence.mjs").evidenceOf>} evidence
 * @param {Record<string, string>} countryNames
 * @returns {Map<number, {name: string, main: boolean}>}
 */
export function namesOf(evidence, countryNames) {
    const totals = {}
    for (const [id, layer] of Object.entries(evidence.layers)) {
        const t = totals[id] = {}
        for (const row of Object.values(layer.byCountry)) for (const [key, n] of Object.entries(row)) t[key] = (t[key] ?? 0) + n
    }

    const candidatesOf = (lm) => {
        const out = []
        LAYERS.forEach(({id, weight, shared}, order) => {
            const layer = evidence.layers[id]
            if (!layer) return
            for (const [key, n] of Object.entries(layer.byLandmass[lm.id] ?? {})) {
                const name = layer.names[key] && cleaned(layer.names[key])
                if (!name) continue
                let fit = 1
                if (id !== "islandPoint") {
                    if (n / lm.tiles < MAJORITY) continue
                    const size = (shared ? totals[id][key] : layer.byCountry[lm.code]?.[key]) ?? n
                    fit = n / (lm.tiles + size - n)
                }
                out.push({name, key: keyOfName(name), order, fit, score: fit * weight})
            }
        })
        return out.sort((a, b) => b.score - a.score || a.order - b.order)
    }

    const byCountry = new Map()
    for (const lm of evidence.landmasses) {
        if (lm.tiles === 0) continue
        if (!byCountry.has(lm.code)) byCountry.set(lm.code, [])
        byCountry.get(lm.code).push(lm)
    }

    const named = new Map()
    for (const [code, list] of byCountry) {
        list.sort((a, b) => b.tiles - a.tiles || a.id - b.id)
        const country = countryNames[code] ?? evidence.countryLabels[code]?.[0] ?? code
        const reserved = new Set([country, ...(evidence.countryLabels[code] ?? [])].map(keyOfName))
        const own = (lm) => candidatesOf(lm).filter((c) => !reserved.has(c.key))

        // Java, not western New Guinea; but Jutland has no name of its own, so Denmark stays on it.
        const capital = list.find((lm) => lm.id === evidence.capitals?.[code])
        const main = capital && own(list[0]).some((c) => c.fit >= CAPITAL_FIT) ? capital : list[0]
        named.set(main.id, {name: country, main: true})

        const taken = new Set(reserved)
        const rest = list.filter((lm) => lm !== main)
        const all = new Map(rest.map((lm) => [lm.id, own(lm)]))
        const pairs = []
        for (const lm of rest) for (const c of all.get(lm.id)) if (c.fit >= FLOOR) pairs.push({lm, c})
        pairs.sort((a, b) => b.c.score - a.c.score || a.c.order - b.c.order || b.lm.tiles - a.lm.tiles || a.lm.id - b.lm.id)
        for (const {lm, c} of pairs) {
            if (named.has(lm.id) || taken.has(c.key)) continue
            named.set(lm.id, {name: c.name, main: false})
            taken.add(c.key)
        }

        for (const lm of rest) {
            if (named.has(lm.id)) continue
            const best = all.get(lm.id)[0]
            if (best && !taken.has(best.key)) {
                named.set(lm.id, {name: best.name, main: false})
                taken.add(best.key)
                continue
            }
            const base = best ? best.name : country
            let n = 2
            while (taken.has(keyOfName(`${base} ${n}`))) n++
            named.set(lm.id, {name: `${base} ${n}`, main: false})
            taken.add(keyOfName(`${base} ${n}`))
        }
    }

    return named
}
