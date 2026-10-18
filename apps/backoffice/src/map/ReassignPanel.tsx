import {useState} from "react"
import {PlanetAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {CountryField} from "../components/CountryField.tsx"
import {countryIdOf, countryLabel} from "../countries.ts"
import {wordsOf} from "../refusal.ts"
import {countOf} from "../support/labels.ts"
import {useAction} from "../useAction.ts"
import {reassign, Reassigned} from "./actions.ts"

type Counted = {from: string, to: string, counts: Reassigned}

// Every tile one country holds goes to another while the game runs. Reassigning back does not undo it: it would
// also move the tiles `to` held before.
export function ReassignPanel({planet}: {planet: PlanetAdmin}) {
    const [from, setFrom] = useState("")
    const [to, setTo] = useState("")
    const [count, start] = useAction<Counted>(wordsOf)

    const pair = {from: countryIdOf(from), to: countryIdOf(to)}
    const ready = pair.from !== "" && pair.to !== ""
    // The question quotes the dry run, so it holds only for the pair it counted.
    const counted = count.step === "done" && count.value.from === pair.from && count.value.to === pair.to
        ? count.value.counts
        : undefined

    return (
        <section className="panel" aria-label="Give a country's tiles to another">
            <p className="hint">
                Nothing undoes it, and reassigning back would also move what the other country held before. Copy
                the table first if you may want to return (<code>deploy/vps/README.md</code>).
            </p>
            <div className="fields">
                <CountryField label="From" value={from} onChange={setFrom}/>
                <CountryField label="To" value={to} onChange={setTo}/>
            </div>
            <div className="action">
                <button
                    type="button"
                    className="button"
                    disabled={!ready || count.step === "running"}
                    onClick={() => start(() => reassign(planet, pair.from, pair.to, true).then(counts => ({...pair, counts})))}
                >
                    Dry run
                </button>
                {count.step === "refused" && <p className="action-refused" role="alert">{count.words}</p>}
                {counted !== undefined && (
                    <p className="action-said" role="status">
                        {countryLabel(pair.from)} holds {countOf(counted.fromBefore, "tile")}, {countryLabel(pair.to)}{" "}
                        {countOf(counted.toBefore, "tile")}.
                    </p>
                )}
            </div>
            <ConfirmAction
                label="Reassign"
                tone="danger"
                disabled={counted === undefined || counted.fromBefore === 0}
                question={`Give ${countryLabel(pair.from)}'s ${countOf(counted?.fromBefore ?? 0, "tile")} to ${countryLabel(pair.to)}?`}
                run={() => reassign(planet, pair.from, pair.to, false)}
                said={done => `Moved ${countOf(done.moved, "tile")}. ${countryLabel(pair.to)} holds ${done.toAfter}.`
                    + (done.fromAfter > 0
                        ? ` ${countryLabel(pair.from)} retook ${done.fromAfter} while it ran: run it again.`
                        : "")}
                refusal={wordsOf}
            />
        </section>
    )
}
