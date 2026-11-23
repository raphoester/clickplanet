import {useState} from "react"
import {SupportAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {Note} from "../gen/grpc/support/v1/admin_pb.ts"
import {Acted, answerNote, closeNote, MAX_ANSWER_LENGTH, refusalOf} from "./actions.ts"

type Props = {
    note: Note
    // Whether the note waits for an answer: only then is there something to close.
    open: boolean
    support: SupportAdmin
    onActed: (acted: Acted) => void
}

// The answer, mailed to the note's address. A refusal keeps the text typed, to send again.
export function AnswerPanel({note, open, support, onActed}: Props) {
    const [text, setText] = useState("")

    return (
        <section className="panel answer" aria-label="Answer">
            <h2>Answer</h2>
            <label className="field">
                <span className="muted">To {note.replyAddress}</span>
                <textarea
                    value={text}
                    maxLength={MAX_ANSWER_LENGTH}
                    rows={6}
                    onChange={event => setText(event.target.value)}
                />
            </label>
            <div className="verdict-buttons">
                <ConfirmAction
                    label="Send"
                    question={`Send this answer to ${note.replyAddress}?`}
                    disabled={text.trim() === ""}
                    run={() => answerNote(support, note, text)}
                    said={acted => acted.said}
                    refusal={err => refusalOf("answer", err)}
                    onDone={onActed}
                />
                {open && (
                    <ConfirmAction
                        label="Close without answering"
                        question="Close this note without answering?"
                        run={() => closeNote(support, note)}
                        said={acted => acted.said}
                        refusal={err => refusalOf("close", err)}
                        onDone={onActed}
                    />
                )}
            </div>
        </section>
    )
}
