import {FormEvent, useId, useState} from "react";
import {countRunes, MAX_NAME_LENGTH, MAX_TEXT_LENGTH} from "../../backends/chat.ts";
import {ChatIdentity, isValidName} from "./chatIdentity.ts";
import {ChatSendFailure} from "./useChat.ts";
import {truncate} from "../truncate.ts";

export type ChatComposerProps = {
    identity: ChatIdentity
    setName: (name: string) => void
    failure?: ChatSendFailure
    onSend: (text: string) => Promise<boolean>
}

const NAME_MAX_LENGTH = 16

const COUNTER_SHOWS_FROM = MAX_TEXT_LENGTH - 40

const FAILURES: Record<ChatSendFailure, string> = {
    'rate-limited': "You're sending messages too fast. Give it a few seconds.",
    'blocked': "This connection is not allowed to post.",
    'rejected': "That message was refused.",
    'failed': "The message could not be sent. Try again.",
}

export default function ChatComposer(props: ChatComposerProps) {
    const [text, setText] = useState("")
    const [draftName, setDraftName] = useState(props.identity.name)
    const [naming, setNaming] = useState(false)
    const nameId = useId()

    const needsName = props.identity.name === "" || naming

    const submitName = (event: FormEvent) => {
        event.preventDefault()
        if (!isValidName(draftName)) return
        props.setName(draftName)
        setNaming(false)
    }

    const submitMessage = async (event: FormEvent) => {
        event.preventDefault()
        const outgoing = text.trim()
        if (outgoing === "" || countRunes(outgoing) > MAX_TEXT_LENGTH) return

        setText("")
        if (!await props.onSend(outgoing)) {
            setText(current => current === "" ? outgoing : current)
        }
    }

    if (needsName) {
        return <form className="chat-composer" onSubmit={submitName}>
            <label className="menu-label" htmlFor={nameId}>Pick a name to chat</label>
            <div className="chat-composer-row">
                <input id={nameId}
                       className="chat-input"
                       value={draftName}
                       autoComplete="off"
                       placeholder="Your name"
                       onChange={e => setDraftName(e.target.value)}/>
                <button type="submit"
                        className="button button-mini chat-send"
                        disabled={!isValidName(draftName)}>
                    OK
                </button>
            </div>
            {countRunes(draftName.trim()) > MAX_NAME_LENGTH &&
                <p className="chat-notice">{MAX_NAME_LENGTH} characters at most.</p>}
        </form>
    }

    const length = countRunes(text.trim())

    return <form className="chat-composer" onSubmit={submitMessage}>
        <div className="chat-composer-row">
            <input className="chat-input"
                   value={text}
                   autoComplete="off"
                   aria-label="Message"
                   placeholder="Say something"
                   onChange={e => setText(e.target.value)}/>
            <button type="submit"
                    className="button button-mini chat-send"
                    disabled={length === 0 || length > MAX_TEXT_LENGTH}>
                Send
            </button>
        </div>

        <div className="chat-composer-foot">
            <span className="menu-label chat-identity">
                as {truncate(props.identity.name, NAME_MAX_LENGTH)}
                <button type="button"
                        className="chat-rename"
                        onClick={() => {
                            setDraftName(props.identity.name)
                            setNaming(true)
                        }}>
                    Change
                </button>
            </span>
            {length >= COUNTER_SHOWS_FROM &&
                <span className={length > MAX_TEXT_LENGTH ? "menu-label chat-counter-over" : "menu-label"}>
                    {MAX_TEXT_LENGTH - length}
                </span>}
        </div>

        {props.failure && <p className="chat-notice" role="alert">{FAILURES[props.failure]}</p>}
    </form>
}
