import {ChatMessage} from "../../backends/chat.ts"
import {PlayerReporter} from "../../backends/support.ts"
import {truncate} from "../truncate.ts"
import {useWordFilter} from "../crazygames/wordFilter.ts"
import NoteModal from "./NoteModal.tsx"

export type ReportMessageModalProps = {
    message: ChatMessage
    reporter: PlayerReporter
    onSent: (message: ChatMessage) => void
    onClose: () => void
}

const NAME_MAX_LENGTH = 24

export default function ReportMessageModal({message, reporter, onSent, onClose}: ReportMessageModalProps) {
    const words = useWordFilter()
    const send = async (text: string) => {
        await reporter.reportPlayer({accountId: message.authorAccountId, reason: "chat-abuse", text, messageId: message.id})
        onSent(message)
    }

    return <NoteModal title="Report a message"
                      label="Details"
                      placeholder="Optional"
                      asked={<blockquote className="support-quoted">
                          <span className="support-quoted-author">{truncate(words.masked(message.authorName), NAME_MAX_LENGTH)}</span>
                          <span className="support-quoted-text">{words.masked(message.text)}</span>
                      </blockquote>}
                      ready
                      optional
                      onSend={send}
                      onClose={onClose}/>
}
