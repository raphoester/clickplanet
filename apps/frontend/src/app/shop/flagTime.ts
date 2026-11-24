import type {FlagTime} from "../../domain/shop.ts"
import {t, translator} from "../../i18n/page.ts"

function dateOf(at: number): string {
    return translator().dateTime(at, {dateStyle: "medium"})
}

// "Renews on Dec 24, 2026", "Ends on …", "Closed on …".
export function flagTimeLine(time: FlagTime): string | undefined {
    switch (time.kind) {
        case "renews":
            return t("shop.renewsOn", {date: dateOf(time.renewal.periodEndsAt)})
        case "stopping":
            return t("shop.endsOn", {date: dateOf(time.renewal.periodEndsAt)})
        case "open":
            return t("shop.endsOn", {date: dateOf(time.endsAt)})
        case "closed":
            return t("shop.closedOn", {date: dateOf(time.closedAt)})
        case "none":
            return undefined
    }
}
