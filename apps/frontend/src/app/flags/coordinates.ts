import type {Pin} from "../../domain/flags.ts"
import type {PinnedPlace} from "./flagFormStore.ts"
import {t, translator} from "../../i18n/page.ts"

export function coordinatesOf({latitude, longitude}: Pin): string {
    const degrees = (value: number) => translator().number(Math.abs(value), {minimumFractionDigits: 2, maximumFractionDigits: 2})
    return t("flagSubmission.coordinates", {
        latitude: degrees(latitude),
        north: latitude >= 0 ? "yes" : "no",
        longitude: degrees(longitude),
        east: longitude >= 0 ? "yes" : "no",
    })
}

export function capitalOf({pin, place}: PinnedPlace): string {
    return place ?? coordinatesOf(pin)
}
