import {Champion} from "../../domain/champion.ts"
import {t, translator} from "../../i18n/page.ts"

export function championNames(champion: Champion): string {
    return translator().list(champion.countryCodes.map((code) => translator().country(code)))
}

export function winsLine(champion: Champion): string {
    return t("season.wins", {count: champion.countryCodes.length, countries: championNames(champion), season: champion.season})
}

export function roundOverLine(round: {number: number, finale: boolean, own: boolean}): string {
    if (!round.own) return t("round.dayOver", {day: round.number})
    return t(round.finale ? "round.finaleOver" : "round.battleOver")
}
