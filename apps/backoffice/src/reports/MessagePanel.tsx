import {Code, ConnectError} from "@connectrpc/connect"
import {useCallback, useState} from "react"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {timeOf} from "../components/time.ts"
import {GetMessageResponse} from "../gen/grpc/chat/v1/admin_pb.ts"
import {ChatAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {useCall} from "../useCall.ts"
import {MESSAGE_GONE, refusalOf, removeMessage} from "./actions.ts"
import {Conversation, Parties} from "./Conversation.tsx"

type Read = {kept: true, message: GetMessageResponse} | {kept: false}

type Props = {
    chat: ChatAdmin
    messageId: string
    parties: Parties
}

export function MessagePanel({chat, messageId, parties}: Props) {
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
            <h2>The message, in its conversation</h2>
            {call.state === "loading" && <p className="muted">Loading…</p>}
            {call.state === "failed" && <FailureNotice failure={call} onRetry={again}/>}
            {call.state === "done" && !call.value.kept && <p className="muted">{MESSAGE_GONE}</p>}
            {call.state === "done" && call.value.kept && (
                <Message
                    chat={chat}
                    message={call.value.message}
                    parties={parties}
                    removedAt={removedAt ?? call.value.message.removedAt?.toDate()}
                    remove={() => removeMessage(chat, messageId)}
                    onRemoved={setRemovedAt}
                />
            )}
        </section>
    )
}

type MessageProps = {
    chat: ChatAdmin
    message: GetMessageResponse
    parties: Parties
    removedAt: Date | undefined
    remove: () => Promise<{removedAt: Date}>
    onRemoved: (at: Date) => void
}

function Message({chat, message, parties, removedAt, remove, onRemoved}: MessageProps) {
    return (
        <>
            <Conversation chat={chat} message={message} removedAt={removedAt} parties={parties}/>
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
