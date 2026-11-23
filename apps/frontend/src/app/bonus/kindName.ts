import {ChargeKind} from "../../domain/bonus.ts"
import {t} from "../../i18n/page.ts"
import {Key} from "../../i18n/translator.ts"

const NAMES: Record<ChargeKind, Key> = {
    refill: "bonus.refill",
    bomb: "bonus.bomb",
    spreadClicks: "bonus.spread",
    encloseClicks: "bonus.enclose",
    shields: "bonus.shield",
    virus: "bonus.virus",
    mine: "bonus.mine",
    rally: "bonus.rally",
    trap: "bonus.trap",
}

// A kind's name, as its slot in the bonus bar says it.
export function kindName(kind: ChargeKind): string {
    return t(NAMES[kind])
}
