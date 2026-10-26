import {useState} from "react"
import {PlanetAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {grant, Grant, Granted, grantsAnything, MAX_GRANT, MOST, NO_GRANT, refusalOf} from "./actions.ts"
import {grantWords, heldWords, wholeOf} from "./charges.ts"

type Props = {planet: PlanetAdmin, accountId: string, who: string}

type Count = "enclosures" | "spreadClicks" | "shields"

const COUNTS: {key: Count, label: string}[] = [
    {key: "enclosures", label: "Enclosures"},
    {key: "spreadClicks", label: "Spread clicks"},
    {key: "shields", label: "Shields"},
]

// Charges as a box gives them. The caps hold, so the answer says what fit; the player sees it at its next page load.
function heldSaid(granted: Granted): string {
    return `It held ${heldWords(granted.before)}. It holds ${heldWords(granted.after)}.`
}

export function GrantPanel({planet, accountId, who}: Props) {
    const [charges, setCharges] = useState<Grant>(NO_GRANT)
    const set = (change: Partial<Grant>) => setCharges(was => ({...was, ...change}))

    return (
        <section className="panel" aria-label="Grant charges">
            <h2>Give {who} charges</h2>
            <div className="checks">
                <label className="check">
                    <input type="checkbox" checked={charges.refill} onChange={e => set({refill: e.target.checked})}/>
                    Refill
                </label>
                <label className="check">
                    <input type="checkbox" checked={charges.bomb} onChange={e => set({bomb: e.target.checked})}/>
                    Bomb
                </label>
            </div>
            <div className="fields">
                {COUNTS.map(({key, label}) => (
                    <label key={key} className="field">
                        <span className="muted">{label}</span>
                        <input
                            type="number"
                            inputMode="numeric"
                            min={0}
                            max={MOST}
                            value={charges[key]}
                            onChange={e => set({[key]: wholeOf(e.target.value)})}
                        />
                    </label>
                ))}
            </div>
            <ConfirmAction
                label="Grant"
                disabled={!grantsAnything(charges)}
                question={`Give ${who} ${grantWords(charges)}?`}
                run={() => grant(planet, accountId, charges)}
                said={heldSaid}
                refusal={err => refusalOf("grant", err)}
            />
            <ConfirmAction
                label="Max out"
                question={`Give ${who} every charge, as many as fit?`}
                run={() => grant(planet, accountId, MAX_GRANT)}
                said={heldSaid}
                refusal={err => refusalOf("grant", err)}
            />
        </section>
    )
}
