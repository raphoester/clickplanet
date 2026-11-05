import {useRef, useState} from "react"
import {PromoAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {Background} from "../gen/grpc/promo/v1/admin_pb.ts"
import {wordsOf} from "../refusal.ts"
import {countOf, sizeLabel} from "../support/labels.ts"
import {Call} from "../useCall.ts"
import {nameOf, percentOf} from "../videos/actions.ts"
import {ACCEPTED, keepBackground, Put, Refused, removeRefusalOf, sendBackground} from "./actions.ts"

type Props = {promo: PromoAdmin, put: Put, library: Call<Background[]>, onChanged: () => void}

export function BackgroundsPanel({promo, put, library, onChanged}: Props) {
    const [picked, setPicked] = useState<{key: number, file: File}[]>([])
    const next = useRef(0)

    return (
        <section className="panel" aria-label="Backgrounds">
            <div className="panel-head">
                <h2>Backgrounds</h2>
                <label className="button button-primary file-pick">
                    Pick images
                    <input
                        type="file"
                        multiple
                        accept={ACCEPTED}
                        onChange={event => {
                            const added = Array.from(event.target.files ?? [], file => ({key: next.current++, file}))
                            setPicked(was => [...was, ...added])
                            event.target.value = ""
                        }}
                    />
                </label>
            </div>
            <p className="muted">A JPEG, a PNG or a WebP of 2 MB at most, shown cropped to a 3:4 portrait card.</p>
            {picked.length > 0 && (
                <ol className="cards promo-grid">
                    {picked.map(({key, file}) => (
                        <PickedImage
                            key={key}
                            promo={promo}
                            put={put}
                            file={file}
                            onKept={onChanged}
                            onClose={() => setPicked(was => was.filter(each => each.key !== key))}
                        />
                    ))}
                </ol>
            )}
            {library.state === "loading" && <p className="muted">Loading…</p>}
            {library.state === "failed" && <FailureNotice failure={library} onRetry={onChanged}/>}
            {library.state === "done" && (
                <>
                    <p className="muted">{countOf(library.value.length, "background")}</p>
                    <ol className="cards promo-grid">
                        {library.value.map(background => (
                            <li key={background.backgroundId} className="card promo-background">
                                <img className="promo-thumb" src={background.url} alt={background.name}/>
                                <strong className="promo-name">{background.name}</strong>
                                <span className="muted">{sizeLabel(Number(background.sizeBytes))}</span>
                                <ConfirmAction
                                    label="Remove"
                                    tone="danger"
                                    question={`Remove the background ${background.name}?`}
                                    run={() => promo.removeBackground({backgroundId: background.backgroundId})}
                                    said={() => "Removed."}
                                    refusal={removeRefusalOf}
                                    onDone={onChanged}
                                />
                            </li>
                        ))}
                    </ol>
                </>
            )}
        </section>
    )
}

type Step =
    | {step: "ready"}
    | {step: "link"}
    | {step: "sending", sent: number}
    | {step: "keeping"}
    | {step: "done"}
    | {step: "refused", words: string}

// An upload asks nothing first, as a video's does: picking the file and pressing Upload is the question.
function PickedImage({promo, put, file, onKept, onClose}: {
    promo: PromoAdmin, put: Put, file: File, onKept: () => void, onClose: () => void,
}) {
    const [name, setName] = useState(() => nameOf(file.name))
    const [step, setStep] = useState<Step>({step: "ready"})
    // The id of the file the bucket holds: a refused name is kept again without sending the file again.
    const [sent, setSent] = useState<string>()
    const busy = step.step === "link" || step.step === "sending" || step.step === "keeping"

    const upload = async () => {
        try {
            let backgroundId = sent
            if (backgroundId === undefined) {
                setStep({step: "link"})
                backgroundId = await sendBackground(promo, put, file, bytes => setStep({step: "sending", sent: bytes}))
                setSent(backgroundId)
            }
            setStep({step: "keeping"})
            await keepBackground(promo, backgroundId, name)
            setStep({step: "done"})
            onKept()
        } catch (err) {
            if (err instanceof Refused && err.resend) setSent(undefined)
            setStep({step: "refused", words: err instanceof Refused ? err.message : wordsOf(err)})
        }
    }

    return (
        <li className="card upload">
            <div className="card-head">
                <strong className="upload-file">{file.name}</strong>
                <span className="muted">{sizeLabel(file.size)}</span>
            </div>
            {step.step === "done" ? (
                <div className="upload-buttons">
                    <p className="action-said" role="status">Uploaded as {name.trim()}.</p>
                    <button type="button" className="button" onClick={onClose}>Close</button>
                </div>
            ) : (
                <>
                    <label className="field">
                        <span className="muted">Name</span>
                        <input value={name} disabled={busy} onChange={event => setName(event.target.value)}/>
                    </label>
                    {step.step === "sending" && (
                        <progress max={file.size} value={step.sent} aria-label={`Uploading ${file.name}`}/>
                    )}
                    <div className="upload-buttons">
                        <button
                            type="button"
                            className="button button-primary"
                            disabled={busy || name.trim() === ""}
                            onClick={upload}
                        >
                            {labelOf(step, file.size)}
                        </button>
                        {!busy && <button type="button" className="button" onClick={onClose}>Cancel</button>}
                    </div>
                    {step.step === "refused" && <p className="action-refused" role="alert">{step.words}</p>}
                </>
            )}
        </li>
    )
}

function labelOf(step: Step, size: number): string {
    switch (step.step) {
        case "link":
            return "Starting…"
        case "sending":
            return `Uploading… ${percentOf(step.sent, size)}%`
        case "keeping":
            return "Keeping…"
        default:
            return "Upload"
    }
}
