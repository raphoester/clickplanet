import {useEffect, useId, useRef, useState} from "react"
import {PlayerReporter, ReportReason} from "../../backends/support.ts"
import {truncate} from "../truncate.ts"
import NoteModal from "./NoteModal.tsx"

export type ReportPlayerModalProps = {
    name: string
    messageId?: string
    reporter: PlayerReporter
    onClose: () => void
}

const NAME_MAX_LENGTH = 24

const REASONS: {reason: ReportReason, label: string}[] = [
    {reason: "cheating", label: "Cheating or a bot"},
    {reason: "chat-abuse", label: "Abuse in the chat"},
    {reason: "bad-name", label: "Offensive name"},
    {reason: "other", label: "Something else"},
]

export default function ReportPlayerModal({name, messageId, reporter, onClose}: ReportPlayerModalProps) {
    const [reason, setReason] = useState<ReportReason>()
    const first = useRef<HTMLButtonElement>(null)
    const labelId = useId()

    useEffect(() => first.current?.focus(), [])

    const send = async (text: string) => {
        if (reason) await reporter.reportPlayer({name, reason, text, messageId})
    }

    return <NoteModal title={`Report ${truncate(name, NAME_MAX_LENGTH)}`}
                      label="Details"
                      placeholder="Optional"
                      asked={<div className="support-reasons">
                          <span className="menu-label" id={labelId}>Why</span>
                          <div className="support-reason-options" role="radiogroup" aria-labelledby={labelId}>
                              {REASONS.map((option, i) => <button key={option.reason}
                                                                  ref={i === 0 ? first : undefined}
                                                                  type="button"
                                                                  role="radio"
                                                                  aria-checked={option.reason === reason}
                                                                  className="panel-box support-reason"
                                                                  onClick={() => setReason(option.reason)}>
                                  {option.label}
                              </button>)}
                          </div>
                      </div>}
                      ready={reason !== undefined}
                      onSend={send}
                      onClose={onClose}/>
}
