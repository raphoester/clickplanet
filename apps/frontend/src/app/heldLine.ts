import {Allegiance, lockOf} from "../domain/allegiance.ts"
import {describeEnd} from "./endTime.ts"
import {t, translator} from "../i18n/page.ts"

export function heldLine(allegiance: Allegiance, now: number): string {
    const country = translator().country(allegiance.countryId)
    const lock = lockOf(allegiance, now)
    if (lock?.kind !== "wait") return t("allegiance.playFor", {country})
    return t("allegiance.playForUntil", {country, time: describeEnd(lock.until, now)})
}
