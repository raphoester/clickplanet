import {useState} from "react"
import {AttachmentUpload, FeedbackBackend, PageContext} from "../../backends/support.ts"
import Choices, {Choice} from "./Choices.tsx"
import NoteModal from "./NoteModal.tsx"

export type FeedbackModalProps = {
    support: FeedbackBackend
    pageContext: () => PageContext
    onClose: () => void
}

type Kind = "idea" | "bug"

type Writing = {label: string, placeholder: string}

const KINDS: Choice<Kind>[] = [
    {value: "idea", label: "An idea"},
    {value: "bug", label: "A bug"},
]

const WRITING: Record<Kind, Writing> = {
    idea: {label: "Your idea", placeholder: "What should the game have?"},
    bug: {label: "What happened", placeholder: "What did you do, and what went wrong?"},
}

const UNPICKED: Writing = {label: "Your message", placeholder: ""}

export default function FeedbackModal({support, pageContext, onClose}: FeedbackModalProps) {
    const [kind, setKind] = useState<Kind>()
    const writing = kind ? WRITING[kind] : UNPICKED

    const send = async (text: string, files: File[]): Promise<AttachmentUpload[]> => {
        switch (kind) {
            case "idea":
                return support.suggestFeature(text, pageContext(), files)
            case "bug":
                return support.reportBug(text, pageContext(), files)
            case undefined:
                return []
        }
    }

    return <NoteModal title="Send feedback"
                      label={writing.label}
                      placeholder={writing.placeholder}
                      asked={<Choices label="It is" options={KINDS} picked={kind} onPick={setKind}/>}
                      ready={kind !== undefined}
                      files={support}
                      onSend={send}
                      onClose={onClose}/>
}
