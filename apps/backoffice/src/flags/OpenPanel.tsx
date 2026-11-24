import {useEffect, useState} from "react"
import {FlagsAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {localOf, timeOf, zoneOf} from "../components/time.ts"
import {sizeLabel} from "../support/labels.ts"
import {percentOf, Put} from "../videos/actions.ts"
import {
    ACCEPTED, daysLater, Draft, draftPin, draftProblems, endOf, leftOf, lengthOf, MAX_NAME, openFlag, openRefusalOf, ready,
    Refused,
} from "./actions.ts"
import {globeLinkOf, pastedPlaceOf, placeWords} from "./pin.ts"

const FIRST_DAYS = 30

const blank = (): Draft => ({
    name: "", file: undefined, endsAt: localOf(daysLater(new Date(), FIRST_DAYS)), latitude: "", longitude: "",
})

// A sold flag opened by hand: its name, its picture and its end. Open asks first, then sends the picture and opens it.
export function OpenPanel({flags, put, onOpened}: {flags: FlagsAdmin, put: Put, onOpened: () => void}) {
    const [draft, setDraft] = useState(blank)
    const [sent, setSent] = useState<number>()
    const change = (part: Partial<Draft>) => setDraft(was => ({...was, ...part}))
    const now = new Date()
    const end = endOf(draft.endsAt)
    const pin = draftPin(draft)
    const {file} = draft
    // A map copies both numbers at once: pasted in either field, each goes to its own.
    const typePin = (typed: string, field: "latitude" | "longitude") => {
        const pasted = pastedPlaceOf(typed)
        change(pasted === undefined ? {[field]: typed} : {latitude: String(pasted.latitude), longitude: String(pasted.longitude)})
    }

    return (
        <section className="panel" aria-label="Open a flag">
            <h2>Open a flag</h2>
            <div className="fields">
                <label className="field">
                    <span className="muted">Name · {lengthOf(draft.name)}/{MAX_NAME}</span>
                    <input value={draft.name} onChange={event => change({name: event.target.value})}/>
                </label>
                <label className="field">
                    <span className="muted">Ends</span>
                    <input type="datetime-local" value={draft.endsAt} onChange={event => change({endsAt: event.target.value})}/>
                </label>
            </div>
            <p className="hint">Times are in {zoneOf()}.{end !== undefined && end > now && ` ${leftOf(end, now)}.`}</p>
            <div className="fields">
                <label className="field">
                    <span className="muted">Pin latitude</span>
                    <input inputMode="decimal" value={draft.latitude} onChange={event => typePin(event.target.value, "latitude")}/>
                </label>
                <label className="field">
                    <span className="muted">Pin longitude</span>
                    <input inputMode="decimal" value={draft.longitude} onChange={event => typePin(event.target.value, "longitude")}/>
                </label>
            </div>
            {pin !== undefined && (
                <p className="hint">
                    <a href={globeLinkOf(pin)} target="_blank" rel="noreferrer">Open {placeWords(pin)} on the globe</a>
                </p>
            )}
            <div className="flag-pick">
                <label className="button file-pick">
                    {file === undefined ? "Pick a picture" : "Pick another picture"}
                    <input
                        type="file"
                        accept={ACCEPTED}
                        onChange={event => {
                            const picked = event.target.files?.[0]
                            if (picked !== undefined) change({file: picked})
                            event.target.value = ""
                        }}
                    />
                </label>
                {file !== undefined && <Picked file={file}/>}
            </div>
            {draftProblems(draft, now).map(problem => <p key={problem} className="action-refused" role="alert">{problem}</p>)}
            {sent !== undefined && file !== undefined && (
                <progress max={file.size} value={sent} aria-label={`Uploading ${file.name}: ${percentOf(sent, file.size)}%`}/>
            )}
            <ConfirmAction
                label="Open"
                disabled={!ready(draft, now)}
                question={end !== undefined && (
                    <>
                        <p>
                            Open <strong>{draft.name.trim()}</strong> until {timeOf(end)} ({leftOf(end, now)})
                            {pin !== undefined && `, pinned at ${placeWords(pin)}`}?
                        </p>
                        <p>
                            Every player sees it at once and may play for it. A flag is never deleted: closed, it stays in
                            this list for good.
                        </p>
                    </>
                )}
                run={async () => {
                    if (file === undefined || end === undefined || pin === undefined) {
                        throw new Refused("Pick a picture, an end and a pin first.")
                    }
                    setSent(0)
                    try {
                        return await openFlag(flags, put, draft.name, file, end, pin, setSent)
                    } finally {
                        setSent(undefined)
                    }
                }}
                said={flag => {
                    const until = flag.sold?.endsAt?.toDate()
                    return `Opened ${flag.name} (${flag.code})${until === undefined ? "" : `, until ${timeOf(until)}`}.`
                }}
                refusal={openRefusalOf}
                onDone={() => {
                    setDraft(blank())
                    onOpened()
                }}
            />
        </section>
    )
}

// The picture as picked, before the server draws it again.
function Picked({file}: {file: File}) {
    const [url, setUrl] = useState<string>()

    useEffect(() => {
        const made = URL.createObjectURL(file)
        setUrl(made)
        return () => URL.revokeObjectURL(made)
    }, [file])

    return (
        <figure className="flag-picked">
            {url !== undefined && <img className="flag-picture-large" src={url} alt={`The flag picked: ${file.name}`}/>}
            <figcaption className="muted">{file.name} · {sizeLabel(file.size)}</figcaption>
        </figure>
    )
}
