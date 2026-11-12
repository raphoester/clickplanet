import {FormEvent, KeyboardEvent, useEffect, useLayoutEffect, useRef, useState} from "react";
import {ChatMessage, countRunes, MAX_TEXT_LENGTH} from "../../backends/chat.ts";
import {GifChoice, GifQuery} from "../../backends/gifs.ts";
import {quoteOf} from "../../domain/chatLog.ts";
import {paintOf} from "../nameplates/nameplate.ts";
import {useNameplate} from "../nameplates/nameplatesStore.ts";
import {describeEnd} from "../endTime.ts";
import {CloseIcon} from "../components/icons.tsx";
import {useEscape} from "../components/useDialog.ts";
import {quoteStyle} from "./authorStyle.ts";
import GifPicker, {GifOffer} from "./GifPicker.tsx";
import QuoteLine from "./QuoteLine.tsx";
import {ChatSendFailure} from "./useRoomLog.ts";
import {useWordFilter} from "../crazygames/wordFilter.ts";
import {t, translator} from "../../i18n/page.ts";
import {Key} from "../../i18n/translator.ts";

export type ChatComposerProps = {
    username?: string
    guestName?: string
    failure?: ChatSendFailure
    mutedUntil?: number
    replyingTo?: ChatMessage
    onCancelReply?: () => void
    onSend: (text: string) => Promise<boolean>
    gifs?: GifOffer
    onSendGif?: (choice: GifChoice, query: GifQuery) => Promise<boolean>
}

const COUNTER_SHOWS_FROM = MAX_TEXT_LENGTH - 40

const FAILURES: Record<ChatSendFailure, Key> = {
    'rate-limited': "chat.failure.rateLimited",
    'blocked': "chat.failure.blocked",
    'muted': "chat.failure.muted",
    'gifs-muted': "chat.failure.gifsMuted",
    'rejected': "chat.failure.rejected",
    'gone': "chat.failure.gone",
    'no-session': "chat.failure.noSession",
    'failed': "chat.failure.failed",
}

export default function ChatComposer(props: ChatComposerProps) {
    const [text, setText] = useState("")
    const [picking, setPicking] = useState(false)
    const box = useRef<HTMLTextAreaElement>(null)
    const words = useWordFilter()

    useLayoutEffect(() => {
        const field = box.current
        if (!field) return
        field.style.height = "auto"
        field.style.height = `${field.scrollHeight + field.offsetHeight - field.clientHeight}px`
    }, [text])

    const {replyingTo} = props
    useEffect(() => {
        if (replyingTo !== undefined) box.current?.focus()
    }, [replyingTo])

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

    const pick = (choice: GifChoice, query: GifQuery) => {
        setPicking(false)
        void props.onSendGif?.(choice, query)
    }

    return <form className="chat-composer" onSubmit={submitMessage}>
        {replyingTo && <Replying to={replyingTo} onCancel={() => props.onCancelReply?.()}/>}

        {picking && props.gifs && <GifPicker offer={props.gifs} onPick={pick} onClose={() => setPicking(false)}/>}

        <div className="chat-composer-row">
            <textarea className="field chat-input"
                      ref={box}
                      rows={1}
                      value={text}
                      autoComplete="off"
                      aria-label={t("chat.composer.message")}
                      placeholder={t("chat.composer.placeholder")}
                      enterKeyHint="send"
                      onKeyDown={sendOnEnter}
                      onChange={e => setText(e.target.value.replace(/[\r\n]+/g, " "))}/>
            {props.gifs && <button type="button"
                                   className="button button-mini button-secondary chat-gif"
                                   aria-label={t("chat.gifs")}
                                   aria-expanded={picking}
                                   onClick={() => setPicking(open => !open)}>
                {t("chat.gif")}
            </button>}
            <button type="submit"
                    className="button button-mini button-action chat-send"
                    disabled={length === 0 || length > MAX_TEXT_LENGTH}>
                {t("common.send")}
            </button>
        </div>

        <div className="chat-composer-foot">
            {props.username !== undefined
                ? <span className="menu-label chat-identity" title={t("chat.composer.renameHint")}>
                    {t("chat.composer.as", {player: words.masked(props.username)})}
                </span>
                : <span className="menu-label chat-identity" title={t("chat.composer.pickNameHint")}>
                    {props.guestName === undefined
                        ? t("chat.composer.asGuest")
                        : t("chat.composer.as", {player: words.masked(props.guestName)})}
                </span>}
            {length >= COUNTER_SHOWS_FROM &&
                <span className={length > MAX_TEXT_LENGTH ? "menu-label chat-counter-over" : "menu-label"}>
                    {translator().number(MAX_TEXT_LENGTH - length)}
                </span>}
        </div>

        {props.failure && <p className="chat-notice" role="alert">{noticeOf(props.failure, props.mutedUntil)}</p>}
    </form>
}

function Replying({to, onCancel}: {to: ChatMessage, onCancel: () => void}) {
    useEscape(onCancel)
    const quote = quoteOf(to)
    const painted = paintOf(useNameplate(quote.authorAccountId))

    return <div className="chat-replying" style={quoteStyle(painted)}>
        <span className="chat-replying-quote"><QuoteLine quote={quote} lead="replyingTo"/></span>
        <button type="button"
                className="chat-replying-cancel"
                aria-label={t("chat.composer.cancelReply")}
                title={t("chat.composer.cancelReply")}
                onClick={onCancel}>
            <CloseIcon size={14}/>
        </button>
    </div>
}

function noticeOf(failure: ChatSendFailure, mutedUntil: number | undefined): string {
    if (mutedUntil === undefined) return t(FAILURES[failure])

    const time = describeEnd(mutedUntil, Date.now())
    if (failure === 'muted') return t("chat.failure.mutedUntil", {time})
    if (failure === 'gifs-muted') return t("chat.failure.gifsMutedUntil", {time})
    return t(FAILURES[failure])
}
