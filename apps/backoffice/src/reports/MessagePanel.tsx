import {Code, ConnectError} from "@connectrpc/connect"
import {useCallback, useState} from "react"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {timeOf} from "../components/time.ts"
import {When} from "../components/When.tsx"
import {GetMessageResponse} from "../gen/grpc/chat/v1/admin_pb.ts"
import {useCall} from "../useCall.ts"
import {ChatAdmin, MESSAGE_GONE, refusalOf, removeMessage} from "./actions.ts"
import {ConfirmAction} from "./ConfirmAction.tsx"

type Read = {kept: true, message: GetMessageResponse} | {kept: false}

export function MessagePanel({chat, messageId}: {chat: ChatAdmin, messageId: string}) {
    const read = useCallback(
        (signal: AbortSignal): Promise<Read> => chat.getMessage({messageId}, {signal}).then(
            message => ({kept: true, message}),
            (err: unknown) => {
                if (ConnectError.from(err).code === Code.NotFound) return {kept: false}
                throw err
            },
        ),
        [chat, messageId],
    )
    const [call, again] = useCall(read)
    const [removedAt, setRemovedAt] = useState<Date | undefined>(undefined)

    return (
        <section className="panel" aria-label="The message">
            <h2>The message</h2>
            {call.state === "loading" && <p className="muted">Loading…</p>}
            {call.state === "failed" && <FailureNotice failure={call} onRetry={again}/>}
            {call.state === "done" && !call.value.kept && <p className="muted">{MESSAGE_GONE}</p>}
            {call.state === "done" && call.value.kept && (
                <Message
                    message={call.value.message}
                    removedAt={removedAt ?? call.value.message.removedAt?.toDate()}
                    remove={() => removeMessage(chat, messageId)}
                    onRemoved={setRemovedAt}
                />
            )}
        </section>
    )
}

type MessageProps = {
    message: GetMessageResponse
    removedAt: Date | undefined
    remove: () => Promise<{removedAt: Date}>
    onRemoved: (at: Date) => void
}

function Message({message, removedAt, remove, onRemoved}: MessageProps) {
    return (
        <>
            <blockquote className={`quote ${removedAt !== undefined ? "quote-removed" : ""}`}>
                <p className="text">{message.text}</p>
                <footer className="muted">
                    <When at={message.sentAt}/>
                    {message.countryId !== "" && <> · {message.countryId.toUpperCase()}</>}
                </footer>
            </blockquote>
            {removedAt !== undefined
                ? <p className="action-said" role="status">Removed at {timeOf(removedAt)}.</p>
                : (
                    <ConfirmAction
                        label="Remove the message"
                        question="Remove this message from the chat, for everyone?"
                        tone="danger"
                        run={remove}
                        said={removed => `Removed at ${timeOf(removed.removedAt)}.`}
                        refusal={err => refusalOf("removeMessage", err)}
                        onDone={removed => onRemoved(removed.removedAt)}
                    />
                )}
        </>
    )
}
