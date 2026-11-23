import {useState} from "react"
import {FlagsAdmin} from "../clients.ts"
import {Copyable} from "../components/AccountId.tsx"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {Fact} from "../components/Fact.tsx"
import {When} from "../components/When.tsx"
import {timeOf} from "../components/time.ts"
import {countOf} from "../support/labels.ts"
import {
    addTime, addTimeRefusalOf, closeFlag, closeRefusalOf, comebackUntil, COMEBACK_DAYS, daysLater, daysOf, leftOf, MAX_DAYS,
    Sold,
} from "./actions.ts"

const FIRST_DAYS = "30"

// One sold flag: its picture, its name and code, open or closed and until when, and the two things done to it.
export function SoldCard({flags, sold, onChanged}: {flags: FlagsAdmin, sold: Sold, onChanged: () => void}) {
    const {flag, picture, endsAt, closedAt} = sold
    const now = new Date()
    const comeback = closedAt === undefined ? undefined : comebackUntil(closedAt)

    return (
        <li className="card sold-flag">
            <img className="flag-picture-large" src={picture} alt={`The flag of ${flag.name}`}/>
            <div className="card-head">
                <h3 className="sold-name">{flag.name}</h3>
                {closedAt === undefined
                    ? <span className="status status-good">Open</span>
                    : <span className="status">Closed</span>}
            </div>
            <dl className="facts">
                <Fact label="Code"><Copyable text={flag.code}/></Fact>
                {closedAt === undefined
                    ? <Fact label="Ends"><When at={endsAt}/> · {leftOf(endsAt, now)}</Fact>
                    : (
                        <>
                            <Fact label="Closed"><When at={closedAt}/></Fact>
                            <Fact label="Comes back">
                                {comeback !== undefined && comeback > now
                                    ? <>with time added until <When at={comeback}/></>
                                    : <span className="muted">no more: closed over {COMEBACK_DAYS} days ago</span>}
                            </Fact>
                        </>
                    )}
            </dl>
            <div className="sold-buttons">
                <AddDays
                    flags={flags}
                    sold={sold}
                    open={closedAt === undefined}
                    late={comeback !== undefined && comeback <= now}
                    onChanged={onChanged}
                />
                <ConfirmAction
                    label="Close"
                    tone="danger"
                    disabled={closedAt !== undefined}
                    question={
                        <>
                            <p>Close <strong>{flag.name}</strong> now?</p>
                            <p>
                                Its players must pick another flag and nobody may click for it; its land stays. In a
                                battle that locks the flags, it plays on until the lock lifts. Time added within{" "}
                                {COMEBACK_DAYS} days opens it again.
                            </p>
                        </>
                    }
                    run={() => closeFlag(flags, flag.code)}
                    said={closed => {
                        const at = closed.sold?.closedAt?.toDate()
                        return at === undefined
                            ? "Closed."
                            : `Closed. Time added until ${timeOf(comebackUntil(at))} opens it again.`
                    }}
                    refusal={closeRefusalOf}
                    onDone={onChanged}
                />
            </div>
        </li>
    )
}

function AddDays({flags, sold, open, late, onChanged}: {
    flags: FlagsAdmin, sold: Sold, open: boolean, late: boolean, onChanged: () => void,
}) {
    const [typed, setTyped] = useState(FIRST_DAYS)
    const days = daysOf(typed)
    const {flag, endsAt} = sold
    const now = new Date()

    return (
        <div className="sold-add">
            <label className="field">
                <span className="muted">Days</span>
                <input
                    type="number"
                    inputMode="numeric"
                    min={1}
                    max={MAX_DAYS}
                    value={typed}
                    disabled={late}
                    onChange={event => setTyped(event.target.value)}
                />
            </label>
            {typed.trim() !== "" && days === undefined && (
                <p className="action-refused" role="alert">A whole number of days, from 1 to {MAX_DAYS}.</p>
            )}
            <ConfirmAction
                label="Add days"
                disabled={days === undefined || late}
                question={days !== undefined && (open
                    ? <p>Add {countOf(days, "day")} to <strong>{flag.name}</strong>? It ends on {timeOf(daysLater(endsAt, days))} instead of {timeOf(endsAt)}.</p>
                    : (
                        <>
                            <p>Open <strong>{flag.name}</strong> again for {countOf(days, "day")}, until {timeOf(daysLater(now, days))}?</p>
                            <p>Every player sees it again at once, with the land it still holds.</p>
                        </>
                    ))}
                run={async () => ({added: await addTime(flags, flag.code, days ?? 0), reopened: !open})}
                said={({added, reopened}) => {
                    const until = added.sold?.endsAt?.toDate()
                    const ends = `It ends ${until === undefined ? "later" : `on ${timeOf(until)}`}.`
                    return reopened ? `Open again. ${ends}` : ends
                }}
                refusal={addTimeRefusalOf}
                onDone={onChanged}
            />
        </div>
    )
}
