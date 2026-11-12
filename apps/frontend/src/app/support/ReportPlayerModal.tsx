import {useState} from "react"
import {PlayerReporter, ReportReason} from "../../backends/support.ts"
import {truncate} from "../truncate.ts"
import {useWordFilter} from "../crazygames/wordFilter.ts"
import Choices from "../components/Choices.tsx"
import NoteModal from "./NoteModal.tsx"
import {Key} from "../../i18n/translator.ts"
import {t} from "../../i18n/page.ts"

export type ReportPlayerModalProps = {
    accountId: string
    // Shown in the title only: the report names the account.
    name: string
    reporter: PlayerReporter
    onClose: () => void
}

const NAME_MAX_LENGTH = 24

const REASONS: {value: ReportReason, label: Key}[] = [
    {value: "cheating", label: "report.reason.cheating"},
    {value: "chat-abuse", label: "report.reason.chatAbuse"},
    {value: "bad-name", label: "report.reason.badName"},
    {value: "other", label: "report.reason.other"},
]

export default function ReportPlayerModal({accountId, name, reporter, onClose}: ReportPlayerModalProps) {
    const [reason, setReason] = useState<ReportReason>()
    const words = useWordFilter()

    const send = async (text: string) => {
        if (reason) await reporter.reportPlayer({accountId, reason, text})
    }

    return <NoteModal title={t("report.playerTitle", {player: truncate(words.masked(name), NAME_MAX_LENGTH)})}
                      label={t("report.details")}
                      placeholder={t("report.optional")}
                      asked={<Choices label={t("report.why")} options={REASONS.map(({value, label}) => ({value, label: t(label)}))} picked={reason} onPick={setReason}/>}
                      ready={reason !== undefined}
                      optional
                      onSend={send}
                      onClose={onClose}/>
}
