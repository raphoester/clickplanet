import {useState} from "react"
import {PlaceReporter, PlaceReportReason} from "../../backends/support.ts"
import Choices, {Choice} from "./Choices.tsx"
import NoteModal from "./NoteModal.tsx"

export type ReportPlaceModalProps = {
    // The tile the player picked in the drawing.
    tileId: number
    reporter: PlaceReporter
    onClose: () => void
}

const REASONS: Choice<PlaceReportReason>[] = [
    {value: "hate", label: "A hateful symbol or words"},
    {value: "sexual", label: "Something sexual"},
    {value: "other", label: "Something else"},
]

export default function ReportPlaceModal({tileId, reporter, onClose}: ReportPlaceModalProps) {
    const [reason, setReason] = useState<PlaceReportReason>()

    const send = async (text: string) => {
        if (reason) await reporter.reportPlace({tileId, reason, text})
    }

    return <NoteModal title="Report a drawing"
                      label="Details"
                      placeholder="Optional"
                      asked={<Choices label="What is it" options={REASONS} picked={reason} onPick={setReason}/>}
                      ready={reason !== undefined}
                      optional
                      onSend={send}
                      onClose={onClose}/>
}
