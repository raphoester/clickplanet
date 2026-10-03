import {ClickBudget, now, tokensAt} from "../../backends/clickBudget.ts"
import {describePrice, factor, percent} from "../../domain/clickPrice.ts"
import {TollStep, tollRows} from "../../domain/toll.ts"
import {useNow} from "../season/useNow.ts"
import {BoltIcon} from "./ClickBudgetMeter.tsx"
import {offerText, SHARED_WITH} from "./clickOffer.ts"
import "./ClicksPanel.css"

export type ClicksPanelProps = {
    budget?: ClickBudget
    countryName: string
    share?: number
    toll: readonly TollStep[]
    onSignIn?: () => void
}

export default function ClicksPanel({budget, countryName, share, toll, onSignIn}: ClicksPanelProps) {
    useNow()

    const price = budget?.price
    const held = price?.share ?? share
    const detail = describePrice(price, countryName)?.detail
    const rows = held === undefined ? [] : tollRows(toll, held)
    const speedUp = onSignIn && budget?.linkedMultiplier

    return <div className="clicks-panel">
        {budget && <div className="panel-box clicks-reading">
            <span className="clicks-number">{Math.floor(tokensAt(budget, now()))}</span>
            <span className="clicks-of">of {budget.capacity}</span>
        </div>}

        {held !== undefined && <p className="clicks-headline">{countryName} holds {percent(held)} of the map</p>}
        {detail && <p className="clicks-detail">{detail}</p>}

        {rows.length > 1 && <table className="clicks-steps">
            <thead>
            <tr>
                <th scope="col">Share of the map</th>
                <th scope="col" className="clicks-steps-refill">Refill</th>
            </tr>
            </thead>
            <tbody>
            {rows.map((row, index) => <tr key={row.share}
                                          className={row.here ? "clicks-step clicks-step--here" : "clicks-step"}
                                          aria-current={row.here ? "true" : undefined}>
                <td className="clicks-steps-share">{index === 0 ? `Under ${percent(rows[1].share)}` : percent(row.share)}</td>
                <td className="clicks-steps-refill">{row.slowdown === 1 ? "Plain rate" : `${factor(row.slowdown)}× slower`}</td>
            </tr>)}
            </tbody>
        </table>}

        {budget?.sharedWith && <p className="clicks-detail">{SHARED_WITH[budget.sharedWith]}</p>}

        {speedUp && <button type="button" className="button button-mini button-action clicks-sign-in" onClick={onSignIn}>
            <BoltIcon/>
            <span>{offerText(budget?.sharedWith, speedUp)}</span>
        </button>}
    </div>
}
