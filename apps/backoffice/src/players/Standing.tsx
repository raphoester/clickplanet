import {timeOf} from "../components/time.ts"
import {Call} from "../useCall.ts"
import {MuteKind} from "../gen/grpc/chat/v1/chat_pb.ts"
import {confirmedWords, Muting, Mutings, muteWordsOf, Reading} from "./actions.ts"

type Props = {reading: Call<Reading>, muting?: Call<Mutings>}

// Where a player stands, at a glance: banned or not, its offences, the jury's opinion, its warning, and muted or not for
// an account, from the whole chat or from its GIFs.
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
            {value.warnedAt !== undefined && (
                <span className="status status-warn">
                    Warned {timeOf(value.warnedAt.toDate())}, {confirmedWords(value.warningAcknowledgedAt?.toDate())}
                </span>
            )}
        </>
    )
}

function Mute({muting}: {muting: Call<Mutings>}) {
    if (muting.state === "loading") return <span className="muted">Reading the chat…</span>
    if (muting.state === "failed") return <span className="muted">The mute could not be read.</span>
    const {chat, gifs} = muting.value
    return (
        <>
            {chat === undefined
                ? <span className="status status-good">Not muted</span>
                : <Muted kind={MuteKind.CHAT} muting={chat}/>}
            {gifs !== undefined && <Muted kind={MuteKind.GIFS} muting={gifs}/>}
        </>
    )
}

function Muted({kind, muting}: {kind: MuteKind, muting: NonNullable<Muting>}) {
    return (
        <span className="status status-bad">
            {muteWordsOf(kind).muted} until {timeOf(muting.until)}{muting.byNetwork && ", by a mute on its network"}
        </span>
    )
}
