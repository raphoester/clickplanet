import {useState} from "react"
import {PlanetAdmin, PlayerAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {timeOf} from "../components/time.ts"
import {countOf} from "../support/labels.ts"
import {useAction} from "../useAction.ts"
import {dateOfInput, localInputOf, Painters, paintersOf, refusalOf, restoreArea} from "./actions.ts"
import {PainterList} from "./PainterList.tsx"

type Props = {
    planet: PlanetAdmin
    player: PlayerAdmin
    tileId: number
    rings: number | undefined
    defaultBefore: Date
}

const refused = (err: unknown) => refusalOf("restore", err)

// The dry run comes first, and the restore asks with its numbers: a restore is undone by nothing.
export function RestorePanel({planet, player, tileId, rings, defaultBefore}: Props) {
    const [beforeValue, setBeforeValue] = useState(localInputOf(defaultBefore))
    const before = dateOfInput(beforeValue)
    const [count, start, reset] = useAction<Painters>(refused)
    const [countedFor, setCountedFor] = useState("")
    const inputs = `${rings}|${beforeValue}`
    const counted = count.step === "done" && countedFor === inputs ? count.value : undefined
    const keptSince = count.step === "done" ? count.value.keptSince : undefined

    const dryRun = () => {
        if (rings === undefined || before === undefined) return
        setCountedFor(inputs)
        start(() => paintersOf(planet, player, tileId, rings, before))
    }

    return (
        <section className="panel" aria-label="Restore">
            <h2>Restore the area</h2>
            <label className="picker">
                <span className="muted">As it was at</span>
                <input
                    type="datetime-local"
                    value={beforeValue}
                    min={keptSince === undefined ? undefined : localInputOf(keptSince)}
                    onChange={event => setBeforeValue(event.target.value)}
                />
            </label>
            <div className="action">
                <button
                    type="button"
                    className="button"
                    disabled={count.step === "running" || rings === undefined || before === undefined}
                    onClick={dryRun}
                >
                    {count.step === "done" ? "Dry run again" : "Dry run"}
                </button>
                {count.step === "refused" && <p className="action-refused" role="alert">{count.words}</p>}
                {counted !== undefined && before !== undefined && (
                    <>
                        <p className="action-said" role="status">
                            {countOf(counted.tiles, "tile")} in the area, {counted.changed} changed since {timeOf(before)}.
                        </p>
                        {counted.keptSince !== undefined && (
                            <p className="hint">The ledger reaches back to {timeOf(counted.keptSince)}.</p>
                        )}
                        <PainterList painters={counted}/>
                    </>
                )}
            </div>
            <ConfirmAction
                label="Restore"
                tone="danger"
                disabled={counted === undefined || counted.changed === 0}
                question={`Put back the ${countOf(counted?.changed ?? 0, "tile")} changed since ${before === undefined ? "then" : timeOf(before)}?`}
                run={() => restoreArea(planet, tileId, rings ?? 0, before ?? defaultBefore, false)}
                said={done => `Restored ${done.restored} of ${countOf(done.changed, "tile")}.`
                    + (done.restored < done.changed ? " The rest were taken again while it ran." : "")}
                refusal={refused}
                onDone={reset}
            />
        </section>
    )
}
