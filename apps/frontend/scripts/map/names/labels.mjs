const GENERIC = new Set([
    "island", "islands", "isla", "islas", "ile", "iles", "isle", "i", "is", "isole", "ilha", "ilhas",
    "archipelago", "arch", "archipielago", "arquipelago", "group", "the", "of", "de", "la", "le", "du", "des",
    "da", "do", "y", "and", "province", "prefecture", "region", "district", "county", "state", "oblast",
])

export const fold = (s) => s.normalize("NFD").replace(/[̀-ͯ]/g, "").toLowerCase()

const tokens = (s) => fold(s).replace(/[‘’ʻ'`.]/g, "").split(/[^a-z0-9]+/).filter((w) => w && !GENERIC.has(w))

function trigrams(s) {
    const out = new Set()
    const t = ` ${s} `
    for (let i = 0; i + 3 <= t.length; i++) out.add(t.slice(i, i + 3))
    return out
}

export function sameThing(a, b) {
    const x = tokens(a).join(" "), y = tokens(b).join(" ")
    if (!x || !y) return true
    if (x.includes(y) || y.includes(x)) return true
    let prefix = 0
    while (prefix < x.length && x[prefix] === y[prefix]) prefix++
    if (prefix >= 4) return true
    const tx = trigrams(x), ty = trigrams(y)
    let both = 0
    for (const g of tx) if (ty.has(g)) both++
    return both / (tx.size + ty.size - both) >= 0.3
}

const ABBREVIATIONS = [
    [/\bIs\.(?=\s|$)/g, "Islands"],
    [/\bI\.(?=\s|$)/g, "Island"],
    [/\bArch\.(?=\s|$)/g, "Archipelago"],
    [/\bPen\.(?=\s|$)/g, "Peninsula"],
    [/\bî\.(?=\s|$)/g, "Île"],
    [/\bN\.(?=\s)/g, "North"],
    [/\bS\.(?=\s)/g, "South"],
    [/\bSt\.(?=\s)/g, "Saint"],
]

export function expanded(label) {
    let s = String(label).replace(/\s+/g, " ").trim()
    if (s.length > 3 && s === s.toUpperCase()) {
        s = s.toLowerCase().replace(/(^|[\s\-(’'])(\p{L})/gu, (_, sep, ch) => sep + ch.toUpperCase())
            .replace(/\b(Of|The|And|De|La|Du|Des)\b/g, (w) => w.toLowerCase())
    }
    for (const [pattern, word] of ABBREVIATIONS) s = s.replace(pattern, word)
    if (/^[a-zà-ÿ]/.test(s)) s = s[0].toUpperCase() + s.slice(1)
    return s
}

const kindOf = (s) => {
    const f = ` ${fold(s)} `
    if (/ (islands|is\.?|archipelago|arch\.?|group|islas|iles|ilhas|isole) /.test(f)) return "many"
    if (/ (island|i\.|isla|ile|ilha|isle) /.test(f)) return "one"
    return null
}

// NAME_EN is Wikidata's and sometimes wrong ("Utupua" → "Ryan Bullard"): trusted only when it names the same thing.
export function englishOf(label, english, others = []) {
    const own = expanded(label)
    if (!english) return own
    const en = String(english).trim()
    if (!sameThing(own, en) && !others.some((o) => o && sameThing(String(o), en))) return own
    const ownWords = new Set(tokens(own))
    const extra = tokens(en).filter((w) => !ownWords.has(w))
    if (extra.length > 0 && tokens(own).length > 0 && tokens(own).every((w) => tokens(en).includes(w))) return own
    const [a, b] = [kindOf(own), kindOf(en)]
    if (a && b && a !== b) return own
    return en
}

const SUFFIX = / (Prefecture|Province|Governorate|Region|District|County|Oblast|Department|Municipality)$/
const COMPASS = /^((north|south|east|west|central|middle|upper|lower|northern|southern|eastern|western|north-?east|north-?west|south-?east|south-?west|northeastern|northwestern|southeastern|southwestern)\s*)+$/i

export const onlyCompass = (name) => COMPASS.test(String(name).trim())

export function adminName(p) {
    const en = String(p.name_en ?? "").trim()
    const raw = en || String(p.name ?? "").trim()
    if (onlyCompass(raw)) {
        const kind = String(p.type_en ?? "").split("|")[0].trim()
        return kind && kind !== "null" ? `${raw} ${kind}` : raw
    }
    const bare = raw.replace(SUFFIX, "")
    return bare.length >= 3 ? bare : raw
}

export const keyOfName = (name) => fold(name).replace(/[^a-z0-9]+/g, " ").trim()
