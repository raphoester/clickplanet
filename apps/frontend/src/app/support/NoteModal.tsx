import {FormEvent, ReactNode, useEffect, useId, useRef, useState} from "react"
import {countRunes} from "../../backends/chat.ts"
import {AttachmentRulesReader, AttachmentUploader, MAX_NOTE_LENGTH, NO_ATTACHMENTS} from "../../backends/support.ts"
import {Refusal, refusalLine, withPicked} from "../../domain/attachments.ts"
import Modal from "../components/Modal.tsx"
import {AttachmentPicker, Going} from "./Attachments.tsx"
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
    // Called once every file is up, with the id of each, in order.
    onSend: (text: string, attachmentIds: string[]) => Promise<void>
    onClose: () => void
}

const COUNTER_SHOWS_FROM = MAX_NOTE_LENGTH - 100

type State =
    | {kind: "writing", failure?: SendFailure, notUploaded?: string}
    | {kind: "uploading", going: Going}
    | {kind: "sending"}
    | {kind: "sent"}

export default function NoteModal(props: NoteModalProps) {
    const [text, setText] = useState("")
    const [state, setState] = useState<State>({kind: "writing"})
    const [rules, setRules] = useState(NO_ATTACHMENTS)
    const [files, setFiles] = useState<File[]>([])
    const [refusal, setRefusal] = useState<Refusal>()
    const uploaded = useRef(new Map<File, string>())
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
    const busy = state.kind === "uploading" || state.kind === "sending"
    const answered = (props.optional === true || length > 0) && (props.asked === undefined || props.ready === true)
    const canSend = !busy && answered && length <= MAX_NOTE_LENGTH

    const pick = (picked: File[]) => {
        const choice = withPicked(files, picked, rules)
        setFiles(choice.files)
        setRefusal(choice.refusal)
    }

    // One file after the other; a file already up is not sent again.
    const uploadAll = async (signal?: AbortSignal): Promise<string[] | undefined> => {
        const ids: string[] = []
        for (const [place, file] of files.entries()) {
            const kept = uploaded.current.get(file)
            if (kept) {
                ids.push(kept)
                continue
            }

            setState({kind: "uploading", going: {place, fraction: 0}})
            try {
                if (!source) throw new Error("this form takes no file")
                const id = await source.uploadAttachment(file, {
                    signal,
                    onProgress: (fraction) => setState({kind: "uploading", going: {place, fraction}}),
                })
                uploaded.current.set(file, id)
                ids.push(id)
            } catch (e) {
                if (signal?.aborted) return undefined
                const failure = sendFailureOf(e)
                if (failure === "failed") console.error("could not upload an attachment", e)
                setState(failure === "failed" ? {kind: "writing", notUploaded: file.name} : {kind: "writing", failure})
                return undefined
            }
        }
        return ids
    }

    const submit = async (event: FormEvent) => {
        event.preventDefault()
        if (!canSend) return

        const signal = closing.current?.signal
        const ids = await uploadAll(signal)
        if (!ids || signal?.aborted) return

        setState({kind: "sending"})
        try {
            await props.onSend(text.trim(), ids)
            setState({kind: "sent"})
        } catch (e) {
            const failure = sendFailureOf(e)
            if (failure === "failed") console.error("could not send to the support service", e)
            setState({kind: "writing", failure})
        }
    }

    if (state.kind === "sent") {
        return <Modal title={props.title}
                      className="modal-support"
                      footer={<button type="button" className="button support-wide" onClick={props.onClose}>Close</button>}
                      onClose={props.onClose}>
            <p className="support-sent" role="status">Sent. Thank you!</p>
        </Modal>
    }

    return <Modal title={props.title}
                  className="modal-support"
                  stayOnBackdropClick={busy || length > 0 || files.length > 0}
                  footer={<button type="submit"
                                  form={formId}
                                  className="button button-action support-wide"
                                  disabled={!canSend}>
                      {state.kind === "uploading" ? `Uploading ${state.going.place + 1} of ${files.length}…` : "Send"}
                  </button>}
                  onClose={props.onClose}>
        <form id={formId} ref={form} className="support-form" onSubmit={submit} aria-busy={busy}>
            {props.asked}
            <label className="menu-label" htmlFor={fieldId}>{props.label}</label>
            <textarea id={fieldId}
                      className={props.optional ? "field support-text support-text--short" : "field support-text"}
                      rows={props.optional ? 3 : 6}
                      value={text}
                      readOnly={busy}
                      placeholder={props.placeholder}
                      onPaste={(e) => {
                          const pasted = Array.from(e.clipboardData.files)
                          if (pasted.length === 0 || rules.maxPerNote === 0 || busy) return

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
                                                       disabled={busy}
                                                       uploaded={(file) => uploaded.current.has(file)}
                                                       going={state.kind === "uploading" ? state.going : undefined}
                                                       onPick={pick}
                                                       onRemove={(place) => {
                                                           setFiles(files.filter((_, n) => n !== place))
                                                           setRefusal(undefined)
                                                       }}/>}
            {refusal && <p className="support-notice" role="alert">{refusalLine(refusal)}</p>}
            {state.kind === "writing" && state.notUploaded &&
                <p className="support-notice" role="alert">{state.notUploaded} could not be uploaded. Try again.</p>}
            {state.kind === "writing" && state.failure &&
                <p className="support-notice" role="alert">{SEND_FAILURES[state.failure]}</p>}
        </form>
    </Modal>
}
