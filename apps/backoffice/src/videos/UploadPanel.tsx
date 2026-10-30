import {useRef, useState} from "react"
import {ClipsAdmin} from "../clients.ts"
import {wordsOf} from "../refusal.ts"
import {sizeLabel} from "../support/labels.ts"
import {ACCEPTED, keepVideo, nameOf, percentOf, Put, Refused, sendFile} from "./actions.ts"

type Picked = {key: number, file: File}

export function UploadPanel({clips, put, onKept}: {clips: ClipsAdmin, put: Put, onKept: () => void}) {
    const [picked, setPicked] = useState<Picked[]>([])
    const next = useRef(0)

    const pick = (files: FileList | null) => {
        const added = Array.from(files ?? [], file => ({key: next.current++, file}))
        setPicked(was => [...was, ...added])
    }

    return (
        <section className="panel" aria-label="Upload">
            <h2>Upload</h2>
            <label className="button button-primary file-pick">
                Pick videos
                <input
                    type="file"
                    multiple
                    accept={ACCEPTED}
                    onChange={event => {
                        pick(event.target.files)
                        event.target.value = ""
                    }}
                />
            </label>
            {picked.length > 0 && (
                <ol className="cards video-grid">
                    {picked.map(({key, file}) => (
                        <PickedFile
                            key={key}
                            clips={clips}
                            put={put}
                            file={file}
                            onKept={onKept}
                            onClose={() => setPicked(was => was.filter(each => each.key !== key))}
                        />
                    ))}
                </ol>
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

function PickedFile({clips, put, file, onKept, onClose}: {
    clips: ClipsAdmin, put: Put, file: File, onKept: () => void, onClose: () => void,
}) {
    const [name, setName] = useState(() => nameOf(file.name))
    const [caption, setCaption] = useState("")
    const [step, setStep] = useState<Step>({step: "ready"})
    // The video id of the file the bucket holds: a refused name is kept again without sending the file again.
    const [sent, setSent] = useState<string>()
    const busy = step.step === "link" || step.step === "sending" || step.step === "keeping"

    const upload = async () => {
        try {
            let videoId = sent
            if (videoId === undefined) {
                setStep({step: "link"})
                videoId = await sendFile(clips, put, file, bytes => setStep({step: "sending", sent: bytes}))
                setSent(videoId)
            }
            setStep({step: "keeping"})
            await keepVideo(clips, videoId, name, caption)
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
                    <label className="field">
                        <span className="muted">Caption</span>
                        <textarea
                            rows={4}
                            value={caption}
                            disabled={busy}
                            onChange={event => setCaption(event.target.value)}
                        />
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
            return "Saving…"
        case "ready":
        case "done":
        case "refused":
            return "Upload"
    }
}
