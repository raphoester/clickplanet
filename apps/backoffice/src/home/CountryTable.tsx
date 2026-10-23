import {useState} from "react"
import {Country} from "../components/Country.tsx"
import {Span, wholeOf} from "./activity.ts"
import {countriesBy, CountryRow, heldOf, Order} from "./ledger.ts"

const SHOWN = 10

const ORDERS: {order: Order, label: string}[] = [
    {order: "held", label: "Held"},
    {order: "won", label: "Won"},
    {order: "lost", label: "Lost"},
]

type Props = {countries: CountryRow[], mapTiles: number, span: Span}

// The bars are a histogram of the column the countries are ordered by.
export function CountryTable({countries, mapTiles, span}: Props) {
    const [order, setOrder] = useState<Order>("held")
    const [all, setAll] = useState(false)

    const sorted = countriesBy(countries, order)
    const shown = all ? sorted : sorted.slice(0, SHOWN)
    const most = Math.max(1, ...countries.map(country => country[order]))
    const held = heldOf(countries)

    return (
        <section className="panel" aria-label="Countries">
            <div className="panel-head">
                <h2>Countries</h2>
                <div className="segmented" role="group" aria-label="Order">
                    {ORDERS.map(item => (
                        <button key={item.order} type="button" aria-pressed={order === item.order}
                                onClick={() => setOrder(item.order)}>
                            {item.label}
                        </button>
                    ))}
                </div>
            </div>
            <p className="hint">
                {wholeOf(held)} of {wholeOf(mapTiles)} tiles held ({shareOf(held, mapTiles)}). Won and lost in the
                last {span.label}.
            </p>
            {countries.length === 0
                ? <p className="muted">Nobody holds a tile.</p>
                : (
                    <table className="table">
                        <thead>
                            <tr>
                                <th scope="col">Country</th>
                                <th scope="col" className="num">Held</th>
                                <th scope="col" className="num">Won</th>
                                <th scope="col" className="num">Lost</th>
                            </tr>
                        </thead>
                        <tbody>
                            {shown.map(country => (
                                <tr key={country.countryId}>
                                    <td>
                                        <Country id={country.countryId}/>
                                        <span className="share" aria-hidden="true">
                                            <span style={{width: `${(100 * country[order]) / most}%`}}/>
                                        </span>
                                    </td>
                                    <td className="num">
                                        {wholeOf(country.held)}{" "}
                                        <span className="muted">{shareOf(country.held, mapTiles)}</span>
                                    </td>
                                    <td className="num">{country.won > 0 ? `+${wholeOf(country.won)}` : "0"}</td>
                                    <td className="num">{country.lost > 0 ? `−${wholeOf(country.lost)}` : "0"}</td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                )}
            {sorted.length > SHOWN && (
                <button type="button" className="button-small more-rows" onClick={() => setAll(shownAll => !shownAll)}>
                    {all ? `Top ${SHOWN}` : `All ${sorted.length}`}
                </button>
            )}
        </section>
    )
}

function shareOf(tiles: number, of: number): string {
    if (of === 0) return "0%"
    const share = (100 * tiles) / of
    return `${share < 10 ? share.toFixed(1) : Math.round(share)}%`
}
