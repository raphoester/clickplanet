import {SharedBy} from "../../backends/clickBudget.ts"
import {factor} from "../../domain/clickPrice.ts"
import {Key} from "../../i18n/translator.ts"
import {t} from "../../i18n/page.ts"

export const SHARED_WITH: Record<SharedBy, Key> = {
    guests: "clicks.sharedWithGuests",
    network: "clicks.sharedWithNetwork",
}

export function offerText(sharedWith: SharedBy | undefined, multiplier: number, fastStart?: boolean): string {
    if (sharedWith === "guests") return t("clicks.signInOwnClicks")
    return fastStart ? t("clicks.signInKeepSpeed") : t("clicks.signInFaster", {factor: factor(multiplier)})
}
