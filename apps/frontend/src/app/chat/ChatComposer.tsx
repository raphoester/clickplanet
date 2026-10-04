import {FormEvent, KeyboardEvent, useLayoutEffect, useRef, useState} from "react";
import {countRunes, MAX_TEXT_LENGTH} from "../../backends/chat.ts";
import {ChatSendFailure} from "./useChat.ts";

export type ChatComposerProps = {
    username?: string
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
    const box = useRef<HTMLTextAreaElement>(null)

    useLayoutEffect(() => {
        const field = box.current
        if (!field) return
        field.style.height = "auto"
        field.style.height = `${field.scrollHeight + field.offsetHeight - field.clientHeight}px`
    }, [text])

    const submitMessage = async (event: FormEvent) => {
        event.preventDefault()
        const outgoing = text.trim()
        if (outgoing === "" || countRunes(outgoing) > MAX_TEXT_LENGTH) return

        setText("")
        if (!await props.onSend(outgoing)) {
            setText(current => current === "" ? outgoing : current)
        }
    }

    // The server drops line breaks, so Enter always sends.
    const sendOnEnter = (event: KeyboardEvent<HTMLTextAreaElement>) => {
        if (event.key !== "Enter" || event.nativeEvent.isComposing) return
        event.preventDefault()
        event.currentTarget.form?.requestSubmit()
    }

    const length = countRunes(text.trim())

    return <form className="chat-composer" onSubmit={submitMessage}>
        <div className="chat-composer-row">
            <textarea className="field chat-input"
                      ref={box}
                      rows={1}
                      value={text}
                      autoComplete="off"
                      aria-label="Message"
                      placeholder="Say something"
                      enterKeyHint="send"
                      onKeyDown={sendOnEnter}
                      onChange={e => setText(e.target.value.replace(/[\r\n]+/g, " "))}/>
            <button type="submit"
                    className="button button-mini button-action chat-send"
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
