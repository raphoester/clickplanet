import type {Story} from "../domain/clip/story.ts"
import type {Speech, Standing, Words} from "./overlay.ts"

type Plural = {other: string} & Partial<Record<Intl.LDMLPluralRule, string>>

// gap: a count of points, "14 POINTS".
type Language = {
    points: Plural
    tiles: Plural
    behind(country: string, gap: string, leader: string): string
    ahead(country: string, gap: string): string
    level(country: string): string
    teasers: {
        behind(country: string, gap: string): string[]
        ahead(country: string, gap: string): string[]
        level(country: string): string[]
    }
    finale(when: string): string
    call(region: string): string
}

const LANGUAGES: Readonly<Record<string, Language>> = {
    en: {
        points: {one: "POINT", other: "POINTS"},
        tiles: {one: "tile", other: "tiles"},
        behind: (country, gap, leader) => `${country} IS ${gap} BEHIND ${leader}`,
        ahead: (country, gap) => `${country} LEADS THE SEASON BY ${gap}`,
        level: (country) => `${country} IS TIED FOR FIRST`,
        teasers: {
            behind: (country, gap) => [`${country} CAN STILL WIN THE SEASON`, `${gap} BETWEEN ${country} AND FIRST PLACE`],
            ahead: (country, gap) => [`CAN ANYONE STOP ${country}?`, `${country} IS ${gap} CLEAR AT THE TOP`],
            level: (country) => [`${country} SHARES FIRST PLACE`, "THE SEASON IS UP FOR GRABS"],
        },
        finale: (when) => `FINAL BATTLE: ${when}`,
        call: () => "YOUR COUNTRY NEEDS YOU",
    },
    fr: {
        points: {one: "POINT", other: "POINTS"},
        tiles: {one: "case", other: "cases"},
        behind: (country, gap) => `${country}\u00a0: ${gap} DE RETARD`,
        ahead: (country, gap) => `${country}\u00a0: ${gap} D’AVANCE`,
        level: (country) => `${country}\u00a0: À ÉGALITÉ EN TÊTE`,
        teasers: {
            behind: (_, gap) => ["LA SAISON N’EST PAS FINIE", `${gap} À RATTRAPER`],
            ahead: (country, gap) => [`EN TÊTE DE LA SAISON\u00a0: ${country}`, `${gap} D’AVANCE À DÉFENDRE`],
            level: () => ["ÉGALITÉ EN TÊTE DE LA SAISON", "TOUT SE JOUE À LA BATAILLE FINALE"],
        },
        finale: (when) => `BATAILLE FINALE\u00a0: ${when}`,
        call: () => "TON PAYS A BESOIN DE TOI",
    },
    es: {
        points: {one: "PUNTO", other: "PUNTOS"},
        tiles: {one: "casilla", other: "casillas"},
        behind: (country, gap) => `${country}: A ${gap} DEL LÍDER`,
        ahead: (country, gap) => `${country}: ${gap} DE VENTAJA`,
        level: (country) => `${country}: EMPATE EN LA CIMA`,
        teasers: {
            behind: (_, gap) => ["LA TEMPORADA NO HA TERMINADO", `${gap} POR REMONTAR`],
            ahead: (country, gap) => [`LÍDER DE LA TEMPORADA: ${country}`, `${gap} DE VENTAJA QUE DEFENDER`],
            level: () => ["EMPATE EN LO MÁS ALTO DE LA TEMPORADA", "TODO SE DECIDE EN LA BATALLA FINAL"],
        },
        finale: (when) => `BATALLA FINAL: ${when}`,
        call: () => "TU PAÍS TE NECESITA",
    },
    pt: {
        points: {one: "PONTO", other: "PONTOS"},
        tiles: {one: "casa", other: "casas"},
        behind: (country, gap) => `${country}: A ${gap} DA LIDERANÇA`,
        ahead: (country, gap) => `${country}: ${gap} DE VANTAGEM`,
        level: (country) => `${country}: EMPATE NA LIDERANÇA`,
        teasers: {
            behind: (_, gap) => ["A TEMPORADA AINDA NÃO ACABOU", `${gap} PARA RECUPERAR`],
            ahead: (country, gap) => [`LÍDER DA TEMPORADA: ${country}`, `${gap} DE VANTAGEM PARA DEFENDER`],
            level: () => ["EMPATE NO TOPO DA TEMPORADA", "TUDO SE DECIDE NA BATALHA FINAL"],
        },
        finale: (when) => `BATALHA FINAL: ${when}`,
        call: (region) => region === "BR" ? "SEU PAÍS PRECISA DE VOCÊ" : "O TEU PAÍS PRECISA DE TI",
    },
    it: {
        points: {one: "PUNTO", other: "PUNTI"},
        tiles: {one: "casella", other: "caselle"},
        behind: (country, gap) => `${country}: A ${gap} DALLA VETTA`,
        ahead: (country, gap) => `${country}: ${gap} DI VANTAGGIO`,
        level: (country) => `${country}: PARI IN VETTA`,
        teasers: {
            behind: (_, gap) => ["LA STAGIONE NON È FINITA", `${gap} DA RECUPERARE`],
            ahead: (country, gap) => [`IN TESTA ALLA STAGIONE: ${country}`, `${gap} DI VANTAGGIO DA DIFENDERE`],
            level: () => ["PARITÀ IN VETTA ALLA STAGIONE", "TUTTO SI DECIDE NELLA BATTAGLIA FINALE"],
        },
        finale: (when) => `BATTAGLIA FINALE: ${when}`,
        call: () => "IL TUO PAESE HA BISOGNO DI TE",
    },
    de: {
        points: {one: "PUNKT", other: "PUNKTE"},
        tiles: {one: "Feld", other: "Felder"},
        behind: (country, gap) => `${country}: ${gap} RÜCKSTAND`,
        ahead: (country, gap) => `${country}: ${gap} VORSPRUNG`,
        level: (country) => `${country}: PUNKTGLEICH AN DER SPITZE`,
        teasers: {
            behind: (_, gap) => ["DIE SAISON IST NOCH NICHT VORBEI", `NOCH ${gap} AUFZUHOLEN`],
            ahead: (country, gap) => [`TABELLENFÜHRER DER SAISON: ${country}`, `${gap} VORSPRUNG ZU VERTEIDIGEN`],
            level: () => ["GLEICHSTAND AN DER SPITZE DER SAISON", "DIE ENTSCHEIDUNG FÄLLT IN DER FINALEN SCHLACHT"],
        },
        finale: (when) => `FINALE SCHLACHT: ${when}`,
        call: () => "DEIN LAND BRAUCHT DICH",
    },
    nl: {
        points: {one: "PUNT", other: "PUNTEN"},
        tiles: {one: "vakje", other: "vakjes"},
        behind: (country, gap) => `${country}: ${gap} ACHTER`,
        ahead: (country, gap) => `${country}: ${gap} VOOR`,
        level: (country) => `${country}: GELIJK AAN DE TOP`,
        teasers: {
            behind: (_, gap) => ["HET SEIZOEN IS NOG NIET VOORBIJ", `NOG ${gap} IN TE HALEN`],
            ahead: (country, gap) => [`LEIDER VAN HET SEIZOEN: ${country}`, `${gap} VOORSPRONG OM TE VERDEDIGEN`],
            level: () => ["GELIJKSPEL AAN DE TOP VAN HET SEIZOEN", "ALLES WORDT BESLIST IN DE FINALE STRIJD"],
        },
        finale: (when) => `FINALE STRIJD: ${when}`,
        call: () => "JE LAND HEEFT JE NODIG",
    },
    pl: {
        points: {one: "PUNKT", few: "PUNKTY", many: "PUNKTÓW", other: "PUNKTU"},
        tiles: {one: "pole", few: "pola", many: "pól", other: "pola"},
        behind: (country, gap) => `${country}: ${gap} STRATY`,
        ahead: (country, gap) => `${country}: ${gap} PRZEWAGI`,
        level: (country) => `${country}: REMIS NA SZCZYCIE`,
        teasers: {
            behind: (_, gap) => ["SEZON JESZCZE SIĘ NIE SKOŃCZYŁ", `${gap} DO ODROBIENIA`],
            ahead: (country, gap) => [`LIDER SEZONU: ${country}`, `${gap} PRZEWAGI DO OBRONY`],
            level: () => ["REMIS NA SZCZYCIE SEZONU", "WSZYSTKO ROZSTRZYGNIE SIĘ W FINAŁOWEJ BITWIE"],
        },
        finale: (when) => `FINAŁOWA BITWA: ${when}`,
        call: () => "TWÓJ KRAJ CIĘ POTRZEBUJE",
    },
    ro: {
        points: {one: "PUNCT", few: "PUNCTE", other: "DE PUNCTE"},
        tiles: {one: "căsuță", few: "căsuțe", other: "de căsuțe"},
        behind: (country, gap) => `${country}: ${gap} ÎN URMĂ`,
        ahead: (country, gap) => `${country}: ${gap} AVANS`,
        level: (country) => `${country}: EGALITATE ÎN FRUNTE`,
        teasers: {
            behind: (_, gap) => ["SEZONUL NU S-A TERMINAT", `${gap} DE RECUPERAT`],
            ahead: (country, gap) => [`LIDERUL SEZONULUI: ${country}`, `${gap} AVANS DE APĂRAT`],
            level: () => ["EGALITATE ÎN FRUNTEA SEZONULUI", "TOTUL SE DECIDE ÎN BĂTĂLIA FINALĂ"],
        },
        finale: (when) => `BĂTĂLIA FINALĂ: ${when}`,
        call: () => "ȚARA TA ARE NEVOIE DE TINE",
    },
    tr: {
        points: {other: "PUAN"},
        tiles: {other: "kare"},
        behind: (country, gap) => `${country}: ${gap} GERİDE`,
        ahead: (country, gap) => `${country}: ${gap} ÖNDE`,
        level: (country) => `${country}: ZİRVEDE EŞİTLİK`,
        teasers: {
            behind: (_, gap) => ["SEZON DAHA BİTMEDİ", `KAPATILACAK FARK: ${gap}`],
            ahead: (country, gap) => [`SEZONUN LİDERİ: ${country}`, `KORUNACAK FARK: ${gap}`],
            level: () => ["SEZONUN ZİRVESİNDE EŞİTLİK", "HER ŞEY FİNAL SAVAŞINDA BELLİ OLACAK"],
        },
        finale: (when) => `FİNAL SAVAŞI: ${when}`,
        call: () => "ÜLKENİN SANA İHTİYACI VAR",
    },
    hu: {
        points: {other: "PONT"},
        tiles: {other: "mező"},
        behind: (country, gap) => `${country}: ${gap} HÁTRÁNY`,
        ahead: (country, gap) => `${country}: ${gap} ELŐNY`,
        level: (country) => `${country}: HOLTVERSENY AZ ÉLEN`,
        teasers: {
            behind: (_, gap) => ["MÉG NEM ÉRT VÉGET A SZEZON", `${gap} HÁTRÁNYT KELL LEDOLGOZNI`],
            ahead: (country, gap) => [`A SZEZON ÉLLOVASA: ${country}`, `${gap} ELŐNYT KELL MEGVÉDENI`],
            level: () => ["HOLTVERSENY A SZEZON ÉLÉN", "MINDEN A VÉGSŐ CSATÁBAN DŐL EL"],
        },
        finale: (when) => `VÉGSŐ CSATA: ${when}`,
        call: () => "A HAZÁDNAK SZÜKSÉGE VAN RÁD",
    },
}

type Locale = Intl.Locale & {getTimeZones?: () => string[], timeZones?: string[]}

export type Voice = {
    language: string
    locale: string
    region: string
    timeZone: string
    zoneNamed: boolean
}

// The country's language as CLDR has it, else English; the time most of its zones read at the finale, named if they disagree.
export function voiceOf(country: string, at: number): Voice {
    const region = country.toUpperCase()
    const likely = new Intl.Locale(`und-${region}`).maximize() as Locale
    const language = likely.language in LANGUAGES ? likely.language : "en"
    const locale = language === likely.language ? `${language}-${region}` : "en-GB"
    const zones = likely.getTimeZones?.() ?? likely.timeZones ?? []
    const read = (timeZone: string) => new Intl.DateTimeFormat("en-GB", {timeStyle: "short", dateStyle: "short", timeZone}).format(at)
    const counts = new Map<string, {zone: string, count: number}>()
    for (const zone of zones) {
        const time = read(zone)
        counts.set(time, {zone: counts.get(time)?.zone ?? zone, count: (counts.get(time)?.count ?? 0) + 1})
    }
    const most = [...counts.values()].reduce<{zone: string, count: number} | undefined>(
        (best, next) => best === undefined || next.count > best.count ? next : best, undefined)
    return {language, locale, region, timeZone: most?.zone ?? "UTC", zoneNamed: counts.size !== 1}
}

// CLDR's "R.-U." and "Congo - Kinshasa" read worse than the game's English names.
function localNameOf(code: string, language: string, english: (code: string) => string): string {
    if (!/^[a-wyz][a-z]$/.test(code)) return english(code)
    for (const style of ["short", "long"] as const) {
        const name = new Intl.DisplayNames([language], {type: "region", style}).of(code.toUpperCase())
        if (name !== undefined && name !== code.toUpperCase() && !/[.(]| - /.test(name)) return name
    }
    return english(code)
}

export type Naming = {
    nameOf(code: string): string
    said(name: string): string
}

const SITE = "https://clickplanet.lol"

export function callUpWordsOf(story: Story, naming: Naming, headline?: string): Words {
    const day = story.day
    const finaleAt = story.finaleAt
    if (day === undefined || finaleAt === undefined) throw new Error("a call-up needs its day and its finale")

    const voice = voiceOf(story.attacker, finaleAt)
    const words = LANGUAGES[voice.language]
    const upper = (text: string) => text.toLocaleUpperCase(voice.locale)
    const numbers = new Intl.NumberFormat(voice.locale)
    const plurals = new Intl.PluralRules(voice.locale)
    const counted = (count: number, plural: Plural) => `${numbers.format(count)} ${plural[plurals.select(count)] ?? plural.other}`
    const english = voice.language === "en"
    const name = (code: string) => upper(english ? naming.nameOf(code) : localNameOf(code, voice.language, naming.nameOf))
    const subject = (code: string) => english ? upper(naming.said(naming.nameOf(code))) : name(code)

    const table = [...day.table].sort((a, b) => b.points - a.points)
    const own = table.find(({country}) => country === story.attacker)
    const other = table.find(({country}) => country === story.victims[0])
    if (own === undefined || other === undefined) throw new Error(`no ${story.attacker} or ${story.victims[0]} in the season's table`)
    const leads = own.points >= other.points
    const points = Math.abs(own.points - other.points)
    const gap = counted(points, words.points)
    const country = subject(story.attacker)

    const when = upper(new Intl.DateTimeFormat(voice.locale, {
        weekday: "long", day: "numeric", month: "long", hour: "numeric", minute: "2-digit", timeZone: voice.timeZone,
        timeZoneName: voice.zoneNamed ? "short" : undefined,
    }).format(finaleAt))
    const clock = new Intl.DateTimeFormat(voice.locale, {
        weekday: "short", day: "numeric", month: "short", hour: "2-digit", minute: "2-digit", timeZone: voice.timeZone,
        timeZoneName: "short",
    })
    const speech: Speech = {
        tiles: (count) => counted(count, words.tiles),
        number: (count) => numbers.format(count),
        clock: (at) => clock.format(at),
    }
    const standing = ({rank, country: code, points: held}: {rank: number, country: string, points: number}): Standing =>
        ({rank, country: code, name: name(code), points: numbers.format(held)})

    return {
        headline: headline ?? (points === 0 ? words.level(country)
            : leads ? words.ahead(country, gap) : words.behind(country, gap, subject(other.country))),
        line: words.finale(when),
        call: words.call(voice.region),
        callFlags: [story.attacker],
        link: `${SITE}/?f=${story.attacker}`,
        teasers: points === 0 ? words.teasers.level(country)
            : leads ? words.teasers.ahead(country, gap) : words.teasers.behind(country, gap),
        table: (leads ? [own, other] : [other, own]).map(standing),
        when: words.finale(when),
        speech,
    }
}
