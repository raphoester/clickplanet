import {timeOf} from "../components/time.ts"
import {Call} from "../useCall.ts"
import {Reading} from "./actions.ts"

// Where a player stands with the antibot, at a glance: banned or not, its offences, the jury's opinion.
export function Standing({reading}: {reading: Call<Reading>}) {
    if (reading.state === "loading") return <span className="muted">Reading the antibot…</span>
    if (reading.state === "failed") return <span className="muted">The antibot could not be read.</span>
    if (!reading.value.on) return <span className="muted">Antibot off</span>
    const value = reading.value.value
    return (
        <span className="standing">
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
        </span>
    )
}
