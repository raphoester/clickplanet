import {useState} from "react"
import {NoteAttachment} from "../gen/grpc/support/v1/admin_pb.ts"
import {mediaOf, sizeLabel} from "./labels.ts"

const NOUNS = {photo: "Photo", video: "Video", file: "File"}

// Each link is good for an hour from the list: past it, the bucket refuses the file and Reload signs new ones.
export function Attachments({files}: {files: NoteAttachment[]}) {
    if (files.length === 0) return null

    return (
        <ul className="attachments" aria-label="Attachments">
            {files.map((file, n) => <Attachment key={file.url} file={file} n={n + 1}/>)}
        </ul>
    )
}

function Attachment({file, n}: {file: NoteAttachment, n: number}) {
    const [failed, setFailed] = useState(false)
    const media = mediaOf(file.contentType)
    const name = `${NOUNS[media]} ${n}`

    return (
        <li className="attachment">
            {!failed && media === "photo" && (
                <a href={file.url} target="_blank" rel="noreferrer">
                    <img src={file.url} alt={name} loading="lazy" onError={() => setFailed(true)}/>
                </a>
            )}
            {!failed && media === "video" && (
                // A player's video has no captions to give it.
                // eslint-disable-next-line jsx-a11y/media-has-caption
                <video
                    src={file.url}
                    aria-label={name}
                    controls
                    playsInline
                    preload="metadata"
                    onError={() => setFailed(true)}
                />
            )}
            {(failed || media === "file") && (
                <p className="attachment-unshown muted">
                    {name} is not shown here: this browser cannot open {file.contentType}, or its link expired
                    (Reload).
                </p>
            )}
            <div className="attachment-foot">
                <span className="muted">{file.contentType} · {sizeLabel(file.sizeBytes)}</span>
                <a href={file.url} target="_blank" rel="noreferrer">Open</a>
            </div>
        </li>
    )
}
