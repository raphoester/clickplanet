import {FormEvent, ReactNode, useEffect, useId, useRef, useState} from "react"
import {countRunes} from "../../backends/chat.ts"
import {
    AttachmentRulesReader,
    AttachmentUpload,
    AttachmentUploader,
    MAX_NOTE_LENGTH,
    NO_ATTACHMENTS,
} from "../../backends/support.ts"
import {Refusal, refusalLine, withPicked} from "../../domain/attachments.ts"
import Modal from "../components/Modal.tsx"
import {AttachmentPicker, Upload, UploadList} from "./Attachments.tsx"
import {SendFailure, sendFailureOf, SEND_FAILURES} from "./sendFailure.ts"
import "./Support.css"

export type NoteFiles = AttachmentRulesReader & AttachmentUploader

export type NoteModalProps = {
    title: string
    label: string
    placeholder: string
    // What the form asks before the text. With it Send waits for `ready`.
    asked?: ReactNode
    ready?: boolean
    optional?: boolean
    // Without it the form takes no file.
    files?: NoteFiles
    // Answers where each file goes, in order; nothing when it takes no file.
    onSend: (text: string, attachments: File[]) => Promise<AttachmentUpload[] | void>
    onClose: () => void
}

const COUNTER_SHOWS_FROM = MAX_NOTE_LENGTH - 100

type State =
    | {kind: "writing", failure?: SendFailure}
    | {kind: "sending"}
    | {kind: "sent", uploads: Upload[], uploading: boolean}

export default function NoteModal(props: NoteModalProps) {
    const [text, setText] = useState("")
    const [state, setState] = useState<State>({kind: "writing"})
    const [rules, setRules] = useState(NO_ATTACHMENTS)
    const [files, setFiles] = useState<File[]>([])
    const [refusal, setRefusal] = useState<Refusal>()
    const tickets = useRef<AttachmentUpload[]>([])
    const closing = useRef<AbortController>()
    const form = useRef<HTMLFormElement>(null)
    const formId = useId()
    const fieldId = useId()
    const source = props.files

    useEffect(() => {
        form.current?.querySelector<HTMLElement>("button, textarea")?.focus()
    }, [])

    // Made here and not once per component: StrictMode runs the cleanup once before the form is ever used.
    useEffect(() => {
        const controller = new AbortController()
        closing.current = controller
        return () => controller.abort()
    }, [])

    useEffect(() => {
        let closed = false
        source?.attachmentRules().then((read) => {
            if (!closed) setRules(read)
        }, (e) => console.error("could not read what a note may carry", e))
        return () => {
            closed = true
        }
    }, [source])

    const length = countRunes(text.trim())
    const sending = state.kind === "sending"
    const answered = (props.optional === true || length > 0) && (props.asked === undefined || props.ready === true)
    const canSend = !sending && answered && length <= MAX_NOTE_LENGTH

    const pick = (picked: File[]) => {
        const choice = withPicked(files, picked, rules)
        setFiles(choice.files)
        setRefusal(choice.refusal)
    }

    const upload = async (from: Upload[]) => {
        const signal = closing.current?.signal
        if (signal?.aborted) return

        let uploads = from.map((sent): Upload => sent.status === "failed" ? {...sent, status: "waiting", fraction: 0} : sent)
        const show = (uploading: boolean) => setState({kind: "sent", uploads, uploading})
        const change = (place: number, changed: Partial<Upload>) => {
            uploads = uploads.map((sent, n) => n === place ? {...sent, ...changed} : sent)
            show(true)
        }

        show(uploads.some((sent) => sent.status !== "done"))
        for (const place of uploads.keys()) {
            if (uploads[place].status === "done") continue

            change(place, {status: "going"})
            try {
                const ticket = tickets.current[place]
                if (!source || !ticket) throw new Error("the server gave no place to upload the file to")

                await source.uploadAttachment(ticket, uploads[place].file, {
                    signal,
                    onProgress: (fraction) => change(place, {fraction}),
                })
                change(place, {status: "done", fraction: 1})
            } catch (e) {
                if (signal?.aborted) return
                console.error("could not upload an attachment", e)
                change(place, {status: "failed"})
            }
        }
        show(false)
    }

    const submit = async (event: FormEvent) => {
        event.preventDefault()
        if (!canSend) return

        setState({kind: "sending"})
        try {
            tickets.current = await props.onSend(text.trim(), files) ?? []
        } catch (e) {
            const failure = sendFailureOf(e)
            if (failure === "failed") console.error("could not send to the support service", e)
            setState({kind: "writing", failure})
            return
        }
        await upload(files.map((file) => ({file, status: "waiting", fraction: 0})))
    }

    if (state.kind === "sent") {
        const failed = state.uploads.filter((sent) => sent.status === "failed").length

        return <Modal title={props.title}
                      className="modal-support"
                      stayOnBackdropClick={state.uploading}
                      footer={<SentFooter uploading={state.uploading}
                                          failed={failed}
                                          onRetry={() => void upload(state.uploads)}
                                          onClose={props.onClose}/>}
                      onClose={props.onClose}>
            <p className="support-sent" role="status">{sentLine(state.uploading, failed)}</p>
            {state.uploads.length > 0 && <UploadList uploads={state.uploads}/>}
        </Modal>
    }

    return <Modal title={props.title}
                  className="modal-support"
                  stayOnBackdropClick={sending || length > 0 || files.length > 0}
                  footer={<button type="submit"
                                  form={formId}
                                  className="button button-action support-wide"
                                  disabled={!canSend}>
                      Send
                  </button>}
                  onClose={props.onClose}>
        <form id={formId} ref={form} className="support-form" onSubmit={submit} aria-busy={sending}>
            {props.asked}
            <label className="menu-label" htmlFor={fieldId}>{props.label}</label>
            <textarea id={fieldId}
                      className={props.optional ? "field support-text support-text--short" : "field support-text"}
                      rows={props.optional ? 3 : 6}
                      value={text}
                      readOnly={sending}
                      placeholder={props.placeholder}
                      onPaste={(e) => {
                          const pasted = Array.from(e.clipboardData.files)
                          if (pasted.length === 0 || rules.maxPerNote === 0 || sending) return

                          e.preventDefault()
                          pick(pasted)
                      }}
                      onChange={(e) => setText(e.target.value)}/>
            {length >= COUNTER_SHOWS_FROM &&
                <span className={length > MAX_NOTE_LENGTH ? "menu-label support-counter support-counter-over" : "menu-label support-counter"}>
                    {MAX_NOTE_LENGTH - length}
                </span>}
            {rules.maxPerNote > 0 && <AttachmentPicker rules={rules}
                                                       files={files}
                                                       disabled={sending}
                                                       onPick={pick}
                                                       onRemove={(place) => {
                                                           setFiles(files.filter((_, n) => n !== place))
                                                           setRefusal(undefined)
                                                       }}/>}
            {refusal && <p className="support-notice" role="alert">{refusalLine(refusal)}</p>}
            {state.kind === "writing" && state.failure &&
                <p className="support-notice" role="alert">{SEND_FAILURES[state.failure]}</p>}
        </form>
    </Modal>
}

function sentLine(uploading: boolean, failed: number): string {
    if (uploading) return "Sent. Uploading your files…"
    if (failed === 1) return "Sent, but 1 file did not upload."
    if (failed > 1) return `Sent, but ${failed} files did not upload.`
    return "Sent. Thank you!"
}

type SentFooterProps = {
    uploading: boolean
    failed: number
    onRetry: () => void
    onClose: () => void
}

function SentFooter({uploading, failed, onRetry, onClose}: SentFooterProps) {
    if (uploading) return <button type="button" className="button support-wide" disabled>Uploading…</button>

    return <div className="support-actions">
        {failed > 0 && <button type="button" className="button button-action support-wide" onClick={onRetry}>Try again</button>}
        <button type="button" className="button support-wide" onClick={onClose}>Close</button>
    </div>
}
