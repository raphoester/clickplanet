import {Country} from "../../domain/countries.ts"
import {ShareAddress, ShareStats, shareUrl} from "../../domain/shareCard.ts"
import {t, translator} from "../../i18n/page.ts"

// The line under the flag on the picture, in capitals.
export function statsLine({rank, tiles}: ShareStats): string {
    const line = rank === null ? t("share.card.noTilesYet") : t("share.card.standing", {rank, tiles})
    return line.toLocaleUpperCase(translator().locale)
}

export function pictureAlt({country, rank, tiles}: ShareStats): string {
    const name = translator().country(country.code)
    return rank === null
        ? t("share.pictureAltNoRank", {country: name})
        : t("share.pictureAlt", {country: name, rank, tiles})
}

export function shareHeadline({country, rank, tiles}: ShareStats): string {
    const name = translator().country(country.code)
    return rank === null
        ? t("share.headlineNoRank", {country: name})
        : t("share.headline", {country: name, rank, tiles})
}

export function shareText(stats: ShareStats, address: ShareAddress): string {
    return `${shareHeadline(stats)} ${shareUrl(address)}`
}

export function breakHeadline(country: Country, nextSeason?: number): string {
    const name = translator().country(country.code)
    return nextSeason === undefined
        ? t("invite.playFor", {country: name})
        : t("invite.playForNextSeason", {country: name, season: nextSeason})
}
