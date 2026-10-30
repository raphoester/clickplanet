import {timeOf} from "../components/time.ts"
import {Call} from "../useCall.ts"
import {Muting, Reading} from "./actions.ts"

type Props = {reading: Call<Reading>, muting?: Call<Muting>}

// Where a player stands, at a glance: banned or not, its offences, the jury's opinion, and muted or not for an account.
export function Standing({reading, muting}: Props) {
    return (
        <span className="standing">
            <Antibot reading={reading}/>
            {muting !== undefined && <Mute muting={muting}/>}
        </span>
    )
}

function Antibot({reading}: {reading: Call<Reading>}) {
    if (reading.state === "loading") return <span className="muted">Reading the antibot…</span>
    if (reading.state === "failed") return <span className="muted">The antibot could not be read.</span>
    if (!reading.value.on) return <span className="muted">Antibot off</span>
    const value = reading.value.value
    return (
        <>
            {value.banned === true
                ? (
                    <span className="status status-bad">
                        Banned until {timeOf(value.bannedUntil?.toDate() ?? new Date())}, offence {value.offence}
                    </span>
                )
                : <span className="status status-good">Not banned</span>}
            {value.guilty === true && <span className="status status-bad">Jury: guilty</span>}
            {value.guilty !== true && value.suspects > 0 && (
                <span className="status status-warn">{value.suspects} of {value.minSuspects} suspects</span>
            )}
            {value.pardonedUntil !== undefined && (
                <span className="status">Pardoned until {timeOf(value.pardonedUntil.toDate())}</span>
            )}
        </>
    )
}

function Mute({muting}: {muting: Call<Muting>}) {
    if (muting.state === "loading") return <span className="muted">Reading the chat…</span>
    if (muting.state === "failed") return <span className="muted">The mute could not be read.</span>
    if (muting.value === undefined) return <span className="status status-good">Not muted</span>
    return (
        <span className="status status-bad">
            Muted until {timeOf(muting.value.until)}{muting.value.byNetwork && ", by a mute on its network"}
        </span>
    )
}
