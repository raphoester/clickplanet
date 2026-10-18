import {PlanetAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {countOf} from "../support/labels.ts"
import {useAction} from "../useAction.ts"
import {refusalOf, revert, Reverted} from "./actions.ts"
import {Caller, isAccount} from "./caller.ts"

type Props = {planet: PlanetAdmin, caller: Caller, who: string}

const refused = (err: unknown) => refusalOf("revert", err)

// The dry run comes first, and the revert asks with its numbers: a revert is undone by nothing.
export function RevertPanel({planet, caller, who}: Props) {
    const [count, start] = useAction<Reverted>(refused)
    const counted = count.step === "done" ? count.value : undefined

    return (
        <section className="panel" aria-label="Revert">
            <h2>Revert {who}</h2>
            <p className="hint">
                Gives each tile {who} still holds back to what it held before. Ban first, or the player paints again
                behind the revert. {isAccount(caller)
                    ? "An account reverts its takes from every address."
                    : "An address reverts the takes of every account behind it."}
            </p>
            <div className="action">
                <button
                    type="button"
                    className="button"
                    disabled={count.step === "running"}
                    onClick={() => start(() => revert(planet, caller, true))}
                >
                    {counted === undefined ? "Dry run" : "Dry run again"}
                </button>
                {count.step === "refused" && <p className="action-refused" role="alert">{count.words}</p>}
                {counted !== undefined && (
                    <p className="action-said" role="status">
                        It took {countOf(counted.touched, "tile")} in the last 72 hours and still holds {counted.held}.
                        Only those change.
                    </p>
                )}
            </div>
            <ConfirmAction
                label="Revert"
                tone="danger"
                disabled={counted === undefined || counted.held === 0}
                question={`Give back the ${countOf(counted?.held ?? 0, "tile")} ${who} holds?`}
                run={() => revert(planet, caller, false)}
                said={done => `Restored ${done.restored} of ${countOf(done.held, "tile")} held.`
                    + (done.restored < done.held ? " The rest were retaken while it ran." : "")}
                refusal={refused}
            />
        </section>
    )
}
