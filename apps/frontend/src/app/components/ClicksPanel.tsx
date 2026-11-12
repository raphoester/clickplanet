import {now, tokensAt} from "../../backends/clickBudget.ts"
import {useClickBudget} from "../clicks/clickBudgetStore.ts"
import {useCountry} from "../countryStore.ts"
import {factor, PriceNote, priceNote, shareShown} from "../../domain/clickPrice.ts"
import {TollStep, tollRows} from "../../domain/toll.ts"
import {t, translator} from "../../i18n/page.ts"
import {rich} from "../../i18n/rich.tsx"
import {useNow} from "../season/useNow.ts"
import {BoltIcon} from "./ClickBudgetMeter.tsx"
import {offerText, SHARED_WITH} from "./clickOffer.ts"
import "./ClicksPanel.css"

export type ClicksPanelProps = {
    share?: number
    toll: readonly TollStep[]
    onSignIn?: () => void
}

export default function ClicksPanel({share, toll, onSignIn}: ClicksPanelProps) {
    const budget = useClickBudget((store) => store.budget)
    const countryCode = useCountry((store) => store.country.code)
    useNow()

    const price = budget?.price
    const held = price?.share ?? share
    const note = priceNote(price)
    const rows = held === undefined ? [] : tollRows(toll, held)
    const speedUp = onSignIn && budget?.linkedMultiplier

    return <div className="clicks-panel">
        {budget && <div className="panel-box clicks-reading">
            {rich("clicks.countOfCapacity", {
                count: <span className="clicks-number">{Math.floor(tokensAt(budget, now()))}</span>,
                capacity: budget.capacity,
                of: (words) => <span className="clicks-of">{words}</span>,
            })}
        </div>}

        {held !== undefined && <p className="clicks-headline">
            {t("clicks.holdsShare", {country: translator().country(countryCode), share: percent(held)})}
        </p>}
        {note && <p className="clicks-detail">{noteText(note)}</p>}

        {rows.length > 1 && <table className="clicks-steps">
            <thead>
            <tr>
                <th scope="col">{t("clicks.shareOfMap")}</th>
                <th scope="col" className="clicks-steps-refill">{t("clicks.refill")}</th>
            </tr>
            </thead>
            <tbody>
            {rows.map((row, index) => <tr key={row.share}
                                          className={row.here ? "clicks-step clicks-step--here" : "clicks-step"}
                                          aria-current={row.here ? "true" : undefined}>
                <td className="clicks-steps-share">{index === 0 ? t("clicks.under", {share: percent(rows[1].share)}) : percent(row.share)}</td>
                <td className="clicks-steps-refill">{row.slowdown === 1 ? t("clicks.plainRate") : t("clicks.slower", {factor: factor(row.slowdown)})}</td>
            </tr>)}
            </tbody>
        </table>}

        {budget?.sharedWith && <p className="clicks-detail">{t(SHARED_WITH[budget.sharedWith])}</p>}

        {speedUp && <button type="button" className="button button-mini button-action clicks-sign-in" onClick={onSignIn}>
            <BoltIcon/>
            <span>{offerText(budget?.sharedWith, speedUp, budget?.fastStart)}</span>
        </button>}
    </div>
}

function noteText(note: PriceNote): string {
    switch (note.kind) {
        case "slower":
            return note.next
                ? t("clicks.refillsSlowerThen", {factor: factor(note.slowdown), next: factor(note.next.slowdown), share: percent(note.next.share)})
                : t("clicks.refillsSlower", {factor: factor(note.slowdown)})
        case "nearing":
            return t("clicks.refillsSlowerAt", {factor: factor(note.next.slowdown), share: percent(note.next.share)})
    }
}

function percent(share: number): string {
    return translator().number(shareShown(share), {style: "percent", maximumFractionDigits: 1})
}
