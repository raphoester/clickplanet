import en from "./locales/en.json"
import landmassNames from "../../static/landmassNames.json"
import englishCountries from "./countries/en.json"
import {Language, SOURCE_LANGUAGE} from "./languages.ts"
import {formatMessage, MissingValueError, Node, parseMessage, Speech, Value} from "./message.ts"
import type {Countdown} from "../domain/seasonClock.ts"

export type Key = keyof typeof en

export type Lines = Readonly<Record<string, string>>

export const ENGLISH: Lines = en

// Natural Earth's names of the landmasses that are not a country's main one, by id, as `npm run landmassNames` writes them.
export const ENGLISH_LANDMASSES: Lines = landmassNames.names

// Each flag's country inside a sentence, keyed `in.<code>`: "in {flag} the United States".
export const ENGLISH_COUNTRIES: Lines = englishCountries

export type Values = Readonly<Record<string, string | number>>

export type RichValues<T> = Readonly<Record<string, Value<T>>>

// The flags ISO has no code for, and the codes CLDR names in a way that reads badly on a board ("Congo - Kinshasa",
// "Saint-Martin (partie néerlandaise)", "État de la Cité du Vatican"): each has its own line, `country.<code>`.
export const OWN_COUNTRY_NAMES: ReadonlySet<string> = new Set(["xb", "xk", "gb-eng", "gb-sct", "gb-wls", "cd", "cg", "sx", "va"])

const COUNTDOWN_UNITS = ["day", "hour", "minute", "second"] as const

const MINUTE_S = 60
const HOUR_S = 60 * MINUTE_S
const DAY_S = 24 * HOUR_S

// What the page says, in one language: its lines, and the browser's Intl in its locale.
export class Translator {
    private readonly parsed = new Map<string, Node[]>()
    private readonly formats = new Map<string, Intl.NumberFormat | Intl.DateTimeFormat>()
    private readonly countries = new Map<string, string>()
    private readonly countriesIn = new Map<string, Node[]>()
    private readonly displayNames = new Map<"short" | "long", Intl.DisplayNames>()
    private readonly cardinals: Intl.PluralRules
    private readonly ordinals: Intl.PluralRules
    private readonly numbers: Intl.NumberFormat
    private readonly speech: Speech

    // locale: the browser's own tag when it speaks the same language ("en-GB" for English), for its dates and digits.
    constructor(
        readonly language: Language,
        private readonly lines: Lines,
        readonly locale: string = language,
        private readonly landmasses: Lines = ENGLISH_LANDMASSES,
        private readonly countryLines: Lines = ENGLISH_COUNTRIES,
    ) {
        this.cardinals = new Intl.PluralRules(locale)
        this.ordinals = new Intl.PluralRules(locale, {type: "ordinal"})
        this.numbers = new Intl.NumberFormat(locale)
        this.speech = {
            number: (n) => this.numbers.format(n),
            category: (n, ordinal) => (ordinal ? this.ordinals : this.cardinals).select(n),
        }
    }

    t(key: Key, values: Values = {}): string {
        return this.parts<never>(key, values).join("")
    }

    // The line with each slot filled as given: an element stays an element, a wrap gets the words inside its tag.
    parts<T>(key: Key, values: RichValues<T> = {}): (string | T)[] {
        try {
            return formatMessage(this.nodesOf(key), values, this.speech)
        } catch (e) {
            if (!(e instanceof MissingValueError) || import.meta.env.DEV) throw e
            console.error(`"${key}" in ${this.language}:`, e)
            return [this.lines[key] ?? ENGLISH[key]]
        }
    }

    number(n: number, options?: Intl.NumberFormatOptions): string {
        return options ? this.numberFormat(options).format(n) : this.numbers.format(n)
    }

    dateTime(at: number | Date, options: Intl.DateTimeFormatOptions): string {
        return this.dateTimeFormat(options).format(at)
    }

    dateTimeRange(from: number | Date, to: number | Date, options: Intl.DateTimeFormatOptions): string {
        return this.dateTimeFormat(options).formatRange(from, to)
    }

    // 1st, 2nd, 3rd; 1er, 2e: the line `format.ordinal` picks the ending by Intl's ordinal category.
    ordinal(rank: number): string {
        return this.t("format.ordinal", {rank})
    }

    list(items: readonly string[], type: Intl.ListFormatType = "conjunction"): string {
        return new Intl.ListFormat(this.locale, {type}).format(items)
    }

    compare(a: string, b: string): number {
        return a.localeCompare(b, this.locale)
    }

    country(code: string): string {
        const known = this.countries.get(code)
        if (known !== undefined) return known
        const name = this.countryNameOf(code)
        this.countries.set(code, name)
        return name
    }

    private countryNameOf(code: string): string {
        if (OWN_COUNTRY_NAMES.has(code)) return this.t(`country.${code}` as Key)
        const region = code.toUpperCase()
        const named = (names: Intl.DisplayNames) => {
            try {
                const name = names.of(region)
                return name === region ? undefined : name
            } catch {
                return undefined
            }
        }
        const short = named(this.regionNames("short"))
        if (short !== undefined && !ABBREVIATED.test(short) && !TWO_PLACES.test(short)) return short
        const long = named(this.regionNames("long"))
        if (long === undefined) return region
        return TWO_PLACES.test(long) ? long.replace(/\s*\([^)]*\)/g, "") : long
    }

    private regionNames(style: "short" | "long"): Intl.DisplayNames {
        const known = this.displayNames.get(style)
        if (known) return known
        const names = new Intl.DisplayNames([this.locale], {type: "region", style, fallback: "none"})
        this.displayNames.set(style, names)
        return names
    }

    // "en {flag} Argentine", "au {flag} Brésil", "w {flag} Argentynie": "in" a country, as the language says it inside
    // a sentence, with its flag where the flag goes.
    countryIn<T>(code: string, flag: T): (string | T)[] {
        return formatMessage(this.countryInNodesOf(code), {flag}, this.speech)
    }

    countryInText(code: string): string {
        return this.countryIn(code, "").join("").replace(/\s+/g, " ").trim()
    }

    private countryInNodesOf(code: string): Node[] {
        const known = this.countriesIn.get(code)
        if (known) return known
        const key = `in.${code}`
        const nodes = parseMessage(this.countryLines[key] ?? ENGLISH_COUNTRIES[key] ?? `{flag} ${this.country(code)}`)
        this.countriesIn.set(code, nodes)
        return nodes
    }

    // A country's main landmass is named after the country.
    landmass(landmass: number, ground: string | undefined): string {
        const id = String(landmass)
        const own = this.landmasses[id] ?? ENGLISH_LANDMASSES[id]
        if (own) return own
        return ground ? this.country(ground) : ""
    }

    // "Europe", "Middle East": a region as regions.json names it, in this language.
    region(region: string): string {
        return this.t(`region.${region.charAt(0).toLowerCase()}${region.slice(1).replace(/ (\w)/g, (_, c: string) => c.toUpperCase())}` as Key)
    }

    // "3 h ago", "now": the time since a moment, in its largest whole unit.
    ago(at: number, now: number): string {
        const seconds = Math.round((at - now) / 1000)
        const format = new Intl.RelativeTimeFormat(this.locale, {numeric: "auto", style: "short"})
        const size = Math.abs(seconds)
        if (size < 45) return format.format(0, "second")
        if (size < HOUR_S) return format.format(Math.round(seconds / MINUTE_S), "minute")
        if (size < DAY_S) return format.format(Math.round(seconds / HOUR_S), "hour")
        return format.format(Math.round(seconds / DAY_S), "day")
    }

    // "2d 05h 03m 09s" from its largest unit that is not 0; `units` keeps that many.
    countdown(countdown: Countdown, units: number = COUNTDOWN_UNITS.length): string {
        const amounts = [countdown.days, countdown.hours, countdown.minutes, countdown.seconds]
        const first = Math.min(amounts.findIndex((amount) => amount > 0), 2)
        const from = first < 0 ? 2 : first
        return COUNTDOWN_UNITS.slice(from, from + units).map((unit, i) => this.numberFormat({
            style: "unit", unit, unitDisplay: "narrow", minimumIntegerDigits: i === 0 && unit !== "minute" ? 1 : 2,
        }).format(amounts[from + i])).join(" ")
    }

    // "2 hours", "90 minutes": a length, in whole hours when it is some.
    length(minutes: number): string {
        return minutes % 60 === 0
            ? this.number(minutes / 60, {style: "unit", unit: "hour", unitDisplay: "long"})
            : this.number(minutes, {style: "unit", unit: "minute", unitDisplay: "long"})
    }

    private nodesOf(key: Key): Node[] {
        const known = this.parsed.get(key)
        if (known) return known
        const nodes = parseMessage(this.lines[key] ?? ENGLISH[key] ?? key)
        this.parsed.set(key, nodes)
        return nodes
    }

    private numberFormat(options: Intl.NumberFormatOptions): Intl.NumberFormat {
        const id = `n${JSON.stringify(options)}`
        const known = this.formats.get(id)
        if (known) return known as Intl.NumberFormat
        const format = new Intl.NumberFormat(this.locale, options)
        this.formats.set(id, format)
        return format
    }

    private dateTimeFormat(options: Intl.DateTimeFormatOptions): Intl.DateTimeFormat {
        const id = `d${JSON.stringify(options)}`
        const known = this.formats.get(id)
        if (known) return known as Intl.DateTimeFormat
        const format = new Intl.DateTimeFormat(this.locale, options)
        this.formats.set(id, format)
        return format
    }
}

// "UK", "R.-U.", "Wlk. Bryt.": a short name that is an abbreviation reads worse than the long one.
const ABBREVIATED = /^[A-Z]{2,3}$|\./u
// "Congo - Kinshasa", "Myanmar (Birmanie)": one name that says two.
const TWO_PLACES = /\(| [-–] /u

export const ENGLISH_TRANSLATOR = new Translator(SOURCE_LANGUAGE, ENGLISH, SOURCE_LANGUAGE, ENGLISH_LANDMASSES, ENGLISH_COUNTRIES)
