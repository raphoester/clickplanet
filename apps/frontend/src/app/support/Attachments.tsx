import {useEffect, useRef, useState} from "react"
import {AttachmentRules} from "../../backends/support.ts"
import {isPhoto, sizeLine} from "../../domain/attachments.ts"
import {CheckIcon, ClipIcon, CloseIcon, FilmIcon} from "../components/icons.tsx"

export type Going = {
    place: number
    fraction: number
}

export type AttachmentPickerProps = {
    rules: AttachmentRules
    files: File[]
    disabled: boolean
    uploaded: (file: File) => boolean
    going?: Going
    onPick: (picked: File[]) => void
    onRemove: (place: number) => void
}

export function AttachmentPicker({rules, files, disabled, uploaded, going, onPick, onRemove}: AttachmentPickerProps) {
    const input = useRef<HTMLInputElement>(null)

    return <div className="support-files">
        {files.length > 0 && <ul className="support-file-list" aria-label="Attached files">
            {files.map((file, place) => <li key={place} className="panel-box support-file">
                <FileThumb file={file}/>
                <span className="support-file-name">{file.name}</span>
                <FileMark file={file} uploaded={uploaded(file)} going={going?.place === place ? going.fraction : undefined}/>
                <button type="button"
                        className="support-file-remove"
                        aria-label={`Remove ${file.name}`}
                        disabled={disabled}
                        onClick={() => onRemove(place)}>
                    <CloseIcon size={14}/>
                </button>
            </li>)}
        </ul>}
        <input ref={input}
               type="file"
               hidden
               multiple
               accept={rules.contentTypes.join(",")}
               onChange={(e) => {
                   onPick(Array.from(e.target.files ?? []))
                   // Or picking the same file again changes nothing, and says nothing.
                   e.target.value = ""
               }}/>
        <button type="button"
                className="button button-mini support-add"
                disabled={disabled || files.length >= rules.maxPerNote}
                onClick={() => input.current?.click()}>
            <ClipIcon/>
            Add photos or videos
        </button>
    </div>
}

function FileMark({file, uploaded, going}: {file: File, uploaded: boolean, going?: number}) {
    if (going !== undefined) {
        const percent = Math.round(going * 100)
        return <>
            <span className="menu-label support-file-mark">{percent}%</span>
            <span className="support-file-track"
                  role="progressbar"
                  aria-label={`Uploading ${file.name}`}
                  aria-valuemin={0}
                  aria-valuemax={100}
                  aria-valuenow={percent}>
                <span style={{width: `${percent}%`}}/>
            </span>
        </>
    }
    if (uploaded) return <span className="support-file-mark support-file-done" role="img" aria-label="Uploaded"><CheckIcon/></span>
    return <span className="menu-label support-file-mark">{sizeLine(file.size)}</span>
}

function FileThumb({file}: {file: File}) {
    const [url, setUrl] = useState<string>()

    useEffect(() => {
        if (!isPhoto(file)) return

        const made = URL.createObjectURL(file)
        setUrl(made)
        return () => URL.revokeObjectURL(made)
    }, [file])

    // A photo this browser cannot draw (HEIC outside Safari) falls back to the plain mark.
    return <span className="support-file-thumb">
        {url ? <img src={url} alt="" onError={() => setUrl(undefined)}/> : isPhoto(file) ? <ClipIcon size={16}/> : <FilmIcon size={18}/>}
    </span>
}
