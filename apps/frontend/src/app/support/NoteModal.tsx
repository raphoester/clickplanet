import {FormEvent, ReactNode, useEffect, useId, useRef, useState} from "react"
import {countRunes} from "../../backends/chat.ts"
import {MAX_NOTE_LENGTH} from "../../backends/support.ts"
import Modal from "../components/Modal.tsx"
import {SendFailure, sendFailureOf, SEND_FAILURES} from "./sendFailure.ts"
import "./Support.css"

export type NoteModalProps = {
    title: string
    label: string
    placeholder: string
    // What the form asks before the text. With it the text is optional, and Send waits for `ready`.
    asked?: ReactNode
    ready?: boolean
    onSend: (text: string) => Promise<void>
    onClose: () => void
}

const COUNTER_SHOWS_FROM = MAX_NOTE_LENGTH - 100

type State = {kind: "writing", failure?: SendFailure} | {kind: "sending"} | {kind: "sent"}

export default function NoteModal(props: NoteModalProps) {
    const [text, setText] = useState("")
    const [state, setState] = useState<State>({kind: "writing"})
    const box = useRef<HTMLTextAreaElement>(null)
    const formId = useId()
    const fieldId = useId()

    const textOnly = props.asked === undefined
    useEffect(() => {
        if (textOnly) box.current?.focus()
    }, [textOnly])

    const length = countRunes(text.trim())
    const sending = state.kind === "sending"
    const answered = textOnly ? length > 0 : props.ready === true
    const canSend = !sending && answered && length <= MAX_NOTE_LENGTH

    const submit = async (event: FormEvent) => {
        event.preventDefault()
        if (!canSend) return

        setState({kind: "sending"})
        try {
            await props.onSend(text.trim())
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
                  stayOnBackdropClick={sending || length > 0}
                  footer={<button type="submit"
                                  form={formId}
                                  className="button button-action support-wide"
                                  disabled={!canSend}>
                      Send
                  </button>}
                  onClose={props.onClose}>
        <form id={formId} className="support-form" onSubmit={submit} aria-busy={sending}>
            {props.asked}
            <label className="menu-label" htmlFor={fieldId}>{props.label}</label>
            <textarea id={fieldId}
                      ref={box}
                      className={textOnly ? "field support-text" : "field support-text support-text--short"}
                      rows={textOnly ? 6 : 3}
                      value={text}
                      readOnly={sending}
                      placeholder={props.placeholder}
                      onChange={(e) => setText(e.target.value)}/>
            {length >= COUNTER_SHOWS_FROM &&
                <span className={length > MAX_NOTE_LENGTH ? "menu-label support-counter support-counter-over" : "menu-label support-counter"}>
                    {MAX_NOTE_LENGTH - length}
                </span>}
            {state.kind === "writing" && state.failure &&
                <p className="support-notice" role="alert">{SEND_FAILURES[state.failure]}</p>}
        </form>
    </Modal>
}
