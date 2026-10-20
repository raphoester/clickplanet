import {useState} from "react"
import {PlanetAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {countOf} from "../support/labels.ts"
import {useAction} from "../useAction.ts"
import {clearArea, Cleared, refusalOf} from "./actions.ts"

type Props = {
    planet: PlanetAdmin
    tileId: number
    rings: number | undefined
}

const refused = (err: unknown) => refusalOf("clear", err)

// For a drawing older than the ledger keeps, which no restore reaches: every tile of the area goes to nobody.
export function ClearPanel({planet, tileId, rings}: Props) {
    const [count, start, reset] = useAction<Cleared>(refused)
    const [countedFor, setCountedFor] = useState<number | undefined>(undefined)
    const counted = count.step === "done" && countedFor === rings ? count.value : undefined

    const dryRun = () => {
        if (rings === undefined) return
        setCountedFor(rings)
        start(() => clearArea(planet, tileId, rings, true))
    }

    return (
        <section className="panel" aria-label="Clear">
            <h2>Clear the area</h2>
            <div className="action">
                <button
                    type="button"
                    className="button"
                    disabled={count.step === "running" || rings === undefined}
                    onClick={dryRun}
                >
                    {count.step === "done" ? "Dry run again" : "Dry run"}
                </button>
                {count.step === "refused" && <p className="action-refused" role="alert">{count.words}</p>}
                {counted !== undefined && (
                    <p className="action-said" role="status">
                        {countOf(counted.tiles, "tile")} in the area, {counted.owned} held by a flag.
                    </p>
                )}
            </div>
            <ConfirmAction
                label="Clear"
                tone="danger"
                disabled={counted === undefined || counted.owned === 0}
                question={`Give the ${countOf(counted?.owned ?? 0, "tile")} held in the area to nobody?`}
                run={() => clearArea(planet, tileId, rings ?? 0, false)}
                said={done => `Cleared ${done.cleared} of ${countOf(done.owned, "tile")} held.`
                    + (done.cleared < done.owned ? " The rest were taken again while it ran." : "")}
                refusal={refused}
                onDone={reset}
            />
        </section>
    )
}
