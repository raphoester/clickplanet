import {useState} from "react"
import {PlayerReporter, ReportReason} from "../../backends/support.ts"
import {truncate} from "../truncate.ts"
import {useWordFilter} from "../crazygames/wordFilter.ts"
import Choices, {Choice} from "../components/Choices.tsx"
import NoteModal from "./NoteModal.tsx"

export type ReportPlayerModalProps = {
    accountId: string
    // Shown in the title only: the report names the account.
    name: string
    reporter: PlayerReporter
    onClose: () => void
}

const NAME_MAX_LENGTH = 24

const REASONS: Choice<ReportReason>[] = [
    {value: "cheating", label: "Cheating or a bot"},
    {value: "chat-abuse", label: "Abuse in the chat"},
    {value: "bad-name", label: "Offensive name"},
    {value: "other", label: "Something else"},
]

export default function ReportPlayerModal({accountId, name, reporter, onClose}: ReportPlayerModalProps) {
    const [reason, setReason] = useState<ReportReason>()
    const words = useWordFilter()

    const send = async (text: string) => {
        if (reason) await reporter.reportPlayer({accountId, reason, text})
    }

    return <NoteModal title={`Report ${truncate(words.masked(name), NAME_MAX_LENGTH)}`}
                      label="Details"
                      placeholder="Optional"
                      asked={<Choices label="Why" options={REASONS} picked={reason} onPick={setReason}/>}
                      ready={reason !== undefined}
                      optional
                      onSend={send}
                      onClose={onClose}/>
}
