import {useState} from "react"
import {FeedbackBackend, PageContext} from "../../backends/support.ts"
import Choices from "../components/Choices.tsx"
import NoteModal from "./NoteModal.tsx"
import {Key} from "../../i18n/translator.ts"
import {t} from "../../i18n/page.ts"

export type FeedbackModalProps = {
    support: FeedbackBackend
    pageContext: () => PageContext
    onClose: () => void
}

type Kind = "idea" | "bug"

type Writing = {label: Key, placeholder?: Key}

const KINDS: {value: Kind, label: Key}[] = [
    {value: "idea", label: "feedback.idea"},
    {value: "bug", label: "feedback.bug"},
]

const WRITING: Record<Kind, Writing> = {
    idea: {label: "feedback.ideaLabel", placeholder: "feedback.ideaPlaceholder"},
    bug: {label: "feedback.bugLabel", placeholder: "feedback.bugPlaceholder"},
}

const UNPICKED: Writing = {label: "feedback.messageLabel"}

export default function FeedbackModal({support, pageContext, onClose}: FeedbackModalProps) {
    const [kind, setKind] = useState<Kind>()
    const writing = kind ? WRITING[kind] : UNPICKED

    const send = async (text: string, attachmentIds: string[]) => {
        switch (kind) {
            case "idea":
                return support.suggestFeature(text, pageContext(), attachmentIds)
            case "bug":
                return support.reportBug(text, pageContext(), attachmentIds)
            case undefined:
                return
        }
    }

    return <NoteModal title={t("feedback.title")}
                      label={t(writing.label)}
                      placeholder={writing.placeholder === undefined ? "" : t(writing.placeholder)}
                      asked={<Choices label={t("feedback.kind")} options={KINDS.map(({value, label}) => ({value, label: t(label)}))} picked={kind} onPick={setKind}/>}
                      ready={kind !== undefined}
                      files={support}
                      onSend={send}
                      onClose={onClose}/>
}
