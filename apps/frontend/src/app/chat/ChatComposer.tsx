import {FormEvent, useState} from "react";
import {countRunes, MAX_TEXT_LENGTH} from "../../backends/chat.ts";
import {ChatSendFailure} from "./useChat.ts";

export type ChatComposerProps = {
    /** A signed-in player's username, which the server posts under. */
    username?: string
    /** The name the server gave a guest's messages, once this tab has posted one. */
    guestName?: string
    failure?: ChatSendFailure
    onSend: (text: string) => Promise<boolean>
}

const COUNTER_SHOWS_FROM = MAX_TEXT_LENGTH - 40

const FAILURES: Record<ChatSendFailure, string> = {
    'rate-limited': "You're sending messages too fast. Give it a few seconds.",
    'blocked': "This connection is not allowed to post.",
    'rejected': "That message was refused.",
    'no-session': "Could not start a session to chat. Try again in a moment.",
    'failed': "The message could not be sent. Try again.",
}

export default function ChatComposer(props: ChatComposerProps) {
    const [text, setText] = useState("")

    const submitMessage = async (event: FormEvent) => {
        event.preventDefault()
        const outgoing = text.trim()
        if (outgoing === "" || countRunes(outgoing) > MAX_TEXT_LENGTH) return

        setText("")
        if (!await props.onSend(outgoing)) {
            setText(current => current === "" ? outgoing : current)
        }
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
            {props.username !== undefined
                ? <span className="menu-label chat-identity" title="Change it in the account menu">
                    as {props.username}
                </span>
                : <span className="menu-label chat-identity" title="Sign in to pick a username">
                    as {props.guestName ?? "a guest"}
                </span>}
            {length >= COUNTER_SHOWS_FROM &&
                <span className={length > MAX_TEXT_LENGTH ? "menu-label chat-counter-over" : "menu-label"}>
                    {MAX_TEXT_LENGTH - length}
                </span>}
        </div>

        {props.failure && <p className="chat-notice" role="alert">{FAILURES[props.failure]}</p>}
    </form>
}
