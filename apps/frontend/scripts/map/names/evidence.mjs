import {countryCodeOf} from "../ground.mjs"
import {clampLon, pointDistance, polygonIndex} from "./polygons.mjs"
import {adminName, englishOf, expanded, onlyCompass} from "./labels.mjs"

// A landmass this small looks for the nearest feature when none holds a tile: tiles are cut at 1:50m, these layers at 1:10m.
const SMALL = 30
const ADMIN_REACH = 0.3
// Half a tile spacing, or Tugidak reads as part of Kodiak.
const GEO_REACH = 0.12
const POINT_REACH = 0.3
const CAPITAL_REACH = 0.5

const OTHER_LANGUAGES = ["NAME_DE", "NAME_FR", "NAME_ES", "NAME_IT", "NAME_NL", "NAME_PT", "NAME_SV", "NAME_PL"]

/**
 * Which Natural Earth feature each tile sits in, per layer, counted per landmass and per country.
 * @param {{count: number, lon: Float64Array, lat: Float64Array, codes: string[], landmass: Uint16Array}} map
 * @param {Record<string, {features: object[]}>} layers Natural Earth's own files, by name
 */
export function evidenceOf(map, layers) {
    const members = Array.from({length: map.codes.length}, () => [])
    for (let t = 0; t < map.count; t++) members[map.landmass[t]].push(t)

    const landmasses = []
    for (let k = 1; k < map.codes.length; k++) landmasses.push({id: k, code: map.codes[k], tiles: members[k].length})

    const countries = layers.ne_50m_admin_0_countries.features
    const adm0Of = new Map()
    const countryLabels = {}
    for (const f of countries) {
        const code = countryCodeOf(f)
        if (!code) continue
        if (!adm0Of.has(code)) adm0Of.set(code, new Set())
        adm0Of.get(code).add(f.properties.ADM0_A3)
        const p = f.properties
        const labels = countryLabels[code] ??= []
        for (const v of [p.NAME, p.NAME_LONG, p.ADMIN, p.NAME_EN, p.BRK_NAME, p.FORMAL_EN, p.NAME_SORT, p.GEOUNIT]) {
            if (v && !labels.includes(String(v))) labels.push(String(v))
        }
    }

    const regions = layers.ne_10m_geography_regions_polys.features
    const admin1 = layers.ne_10m_admin_1_states_provinces.features
    const ofClass = (classes) => regions.filter((f) => classes.includes(f.properties.FEATURECLA))
    const geoName = (p) => englishOf(p.NAME, p.NAME_EN, OTHER_LANGUAGES.map((k) => p[k]))
    const subunitName = (p) => expanded(p.NAME_LONG)

    const polygonLayers = [
        {id: "subunit50", features: layers.ne_50m_admin_0_map_subunits.features, name: subunitName, country: (p) => p.ADM0_A3, reach: ADMIN_REACH},
        {id: "subunit10", features: layers.ne_10m_admin_0_map_subunits.features, name: subunitName, country: (p) => p.ADM0_A3, reach: ADMIN_REACH},
        {id: "island", features: ofClass(["Island"]), name: geoName, alias: (p) => p.NAME_EN, reach: GEO_REACH},
        {id: "islandGroup", features: ofClass(["Island group"]), name: geoName, alias: (p) => p.NAME_EN, reach: GEO_REACH},
        {id: "landform", features: ofClass(["Peninsula", "Pen/cape", "Isthmus", "Delta"]), name: geoName, alias: (p) => p.NAME_EN, reach: GEO_REACH},
        {id: "admin1", features: admin1, name: adminName, country: (p) => p.adm0_a3, reach: ADMIN_REACH},
    ]

    const evidence = {landmasses, countryLabels, layers: {}}
    const admin1Hits = new Map()
    const bump = (table, a, key) => {
        const row = table[a] ??= {}
        row[key] = (row[key] ?? 0) + 1
    }

    for (const layer of polygonLayers) {
        const index = polygonIndex(layer.features)
        const names = layer.features.map((f) => layer.name(f.properties))
        // Two polygons under one label ("Lyakhov Islands" twice) go by their English labels when those differ.
        if (layer.alias) {
            const byName = new Map()
            names.forEach((n, f) => byName.set(n, [...(byName.get(n) ?? []), f]))
            for (const same of byName.values()) {
                const aliases = same.map((f) => layer.alias(layer.features[f].properties))
                if (same.length > 1 && aliases.every(Boolean) && new Set(aliases).size === same.length) {
                    same.forEach((f, i) => names[f] = aliases[i])
                }
            }
        }

        const byCountry = {}, byLandmass = {}
        for (let t = 0; t < map.count; t++) {
            const k = map.landmass[t]
            if (k === 0) continue
            const code = map.codes[k]
            const own = adm0Of.get(code) ?? new Set()
            const accept = layer.country ? (f) => own.has(layer.country(layer.features[f].properties)) : () => true
            const x = clampLon(map.lon[t]), y = map.lat[t]
            let hits = index.contains(x, y).filter(accept)
            if (hits.length === 0 && members[k].length <= SMALL) {
                const near = index.nearest(x, y, layer.reach, accept)
                if (near) hits = [near.f]
            }
            for (const f of hits) {
                bump(byCountry, code, f)
                bump(byLandmass, k, f)
            }
            if (layer.id === "admin1") admin1Hits.set(t, hits)
        }
        const used = new Set(Object.values(byCountry).flatMap((row) => Object.keys(row)))
        evidence.layers[layer.id] = {names: Object.fromEntries([...used].map((f) => [f, names[f]])), byCountry, byLandmass}
    }

    // The unit above admin-1 where Natural Earth records it: Corse, Kyushu.
    {
        const byCountry = {}, byLandmass = {}, names = {}
        for (const [t, hits] of admin1Hits) {
            const k = map.landmass[t]
            for (const f of hits) {
                const region = String(admin1[f].properties.region ?? "").trim()
                if (!region || region === "null" || onlyCompass(region)) continue
                const key = `${admin1[f].properties.adm0_a3}|${region}`
                names[key] = region
                bump(byCountry, map.codes[k], key)
                bump(byLandmass, k, key)
            }
        }
        evidence.layers.admin1Region = {names, byCountry, byLandmass}
    }

    const near = nearestTiles(map)

    {
        const byLandmass = {}, names = {}
        layers.ne_10m_geography_regions_points.features.forEach((f, i) => {
            const p = f.properties
            if (!["island", "island group"].includes(p.featurecla)) return
            const [lon, lat] = f.geometry.coordinates
            const tile = near(lon, lat, POINT_REACH)
            if (tile < 0) return
            names[i] = englishOf(p.name, p.name_en, ["name_de", "name_fr", "name_es"].map((key) => p[key]))
            bump(byLandmass, map.landmass[tile], i)
        })
        evidence.layers.islandPoint = {names, byCountry: {}, byLandmass}
    }

    const codeOfAdm0 = new Map()
    for (const [code, set] of adm0Of) for (const a3 of set) codeOfAdm0.set(a3, code)
    const capitals = {}
    for (const kind of ["Admin-0 capital", "Admin-0 region capital"]) {
        for (const f of layers.ne_10m_populated_places_simple.features) {
            const p = f.properties
            if (p.featurecla !== kind) continue
            const code = codeOfAdm0.get(p.adm0_a3) ?? String(p.iso_a2 ?? "").toLowerCase()
            if (!code || capitals[code]) continue
            const [lon, lat] = f.geometry.coordinates
            const tile = near(lon, lat, CAPITAL_REACH, (t) => map.codes[map.landmass[t]] === code)
            if (tile >= 0) capitals[code] = map.landmass[tile]
        }
    }
    evidence.capitals = capitals

    return evidence
}

function nearestTiles(map) {
    const grid = new Map()
    for (let t = 0; t < map.count; t++) {
        if (!map.landmass[t]) continue
        const key = `${Math.floor(map.lon[t])},${Math.floor(map.lat[t])}`
        if (!grid.has(key)) grid.set(key, [])
        grid.get(key).push(t)
    }

    return (lon, lat, reach, accept = () => true) => {
        let best = -1, bestD = reach
        for (let dx = -1; dx <= 1; dx++) for (let dy = -1; dy <= 1; dy++) {
            for (const t of grid.get(`${Math.floor(lon) + dx},${Math.floor(lat) + dy}`) ?? []) {
                if (!accept(t)) continue
                const d = pointDistance(lon, lat, map.lon[t], map.lat[t])
                if (d < bestD) {
                    bestD = d
                    best = t
                }
            }
        }
        return best
    }
}
