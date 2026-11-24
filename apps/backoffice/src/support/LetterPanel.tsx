import {useCallback, useEffect, useState} from "react"
import {SupportAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {Letter} from "../gen/grpc/support/v1/admin_pb.ts"
import {useCall} from "../useCall.ts"
import {MAX_ANSWER_LENGTH} from "./actions.ts"
import {closeLetter, letterRefusalOf, threadOf, waitsOf, writeLetter, Writing} from "./letters.ts"
import {LineHead} from "./NoteDetail.tsx"

type Props = {
    support: SupportAdmin
    submissionId: string
    accountId: string
    // The player as the page names it.
    name: string
    // The letter's subject, set by the page.
    subject: string
    // A letter sent or closed: what waits for the operator changed.
    onChanged: () => void
}

// The one letter about a submission: its thread, then one more mail in it, or its first.
export function LetterPanel({support, submissionId, accountId, name, subject, onChanged}: Props) {
    const read = useCallback(
        async (signal: AbortSignal) => (await support.listLetters({submissionId, limit: 1}, {signal})).letters,
        [support, submissionId],
    )
    const [call, again] = useCall(read)
    const [letters, setLetters] = useState<Letter[]>()
    useEffect(() => {
        if (call.state === "done") setLetters(call.value)
    }, [call])

    const kept = (letter: Letter) => {
        setLetters([letter])
        onChanged()
    }
    const letter = letters?.at(0)
    const writing: Writing = letter === undefined ? {submissionId, accountId, subject} : {letterId: letter.id}

    return (
        <section className="panel letters" aria-label={`Write to ${name}`}>
            <h2>Write to {name}</h2>
            {call.state === "failed" && <FailureNotice failure={call} onRetry={again}/>}
            {letters === undefined && call.state === "loading" && <p className="muted">Loading the letter…</p>}
            {letters !== undefined && (
                <article className="letter" aria-label={`Letter: ${letter?.subject ?? subject}`}>
                    {letter !== undefined && <Thread letter={letter} name={name}/>}
                    {letter?.address !== "" && (
                        <Composer
                            support={support}
                            writing={writing}
                            to={letter?.address ?? `${name}'s verified address`}
                            label={letter === undefined ? "Text" : "Write again"}
                            waiting={letter !== undefined && waitsOf(letter) ? letter : undefined}
                            onKept={kept}
                        />
                    )}
                </article>
            )}
        </section>
    )
}

function Thread({letter, name}: {letter: Letter, name: string}) {
    return (
        <>
            <div className="card-head">
                <h3 className="letter-subject">{letter.subject}</h3>
                {waitsOf(letter) && <span className="badge badge-replied">Replied</span>}
            </div>
            <p className="muted">{letter.address === "" ? "Its address is no longer kept." : `To ${letter.address}`}</p>
            <ol className="lines" aria-label="What was written">
                {threadOf(letter).map((entry, n) => {
                    switch (entry.kind) {
                        case "mail":
                            return (
                                <li key={n} className="line line-answer">
                                    <LineHead who="ClickPlanet" at={entry.at}/>
                                    <p className="line-text">{entry.text}</p>
                                </li>
                            )
                        case "reply":
                            return (
                                <li key={n} className="line">
                                    <LineHead who={entry.from !== "" && entry.from !== letter.address ? entry.from : name} at={entry.at}/>
                                    <p className="line-text">{entry.text}</p>
                                </li>
                            )
                        case "closed":
                            return (
                                <li key={n} className="line line-closed">
                                    <LineHead who="Closed with no further mail" at={entry.at}/>
                                </li>
                            )
                    }
                })}
            </ol>
        </>
    )
}

// The same before and after the first mail, so what the server answered stays in view.
function Composer({support, writing, to, label, waiting, onKept}: {
    support: SupportAdmin,
    writing: Writing,
    to: string,
    label: string,
    // The letter whose reply waits, to close.
    waiting: Letter | undefined,
    onKept: (letter: Letter) => void,
}) {
    const [text, setText] = useState("")

    return (
        <>
            <label className="field">
                <span className="muted">{label}</span>
                <textarea value={text} maxLength={MAX_ANSWER_LENGTH} rows={4} onChange={event => setText(event.target.value)}/>
            </label>
            <div className="verdict-buttons">
                <ConfirmAction
                    label="Send"
                    disabled={text.trim() === ""}
                    question={`Send this to ${to}?`}
                    run={() => writeLetter(support, writing, text)}
                    said={sent => `Sent to ${sent.address}.`}
                    refusal={err => letterRefusalOf("write", err)}
                    onDone={sent => {
                        setText("")
                        onKept(sent)
                    }}
                />
                {waiting !== undefined && (
                    <ConfirmAction
                        label="Close"
                        question="Close this letter with no further mail? It waits for nothing until the player writes again."
                        run={() => closeLetter(support, waiting)}
                        said={() => "Closed."}
                        refusal={err => letterRefusalOf("close", err)}
                        onDone={onKept}
                    />
                )}
            </div>
        </>
    )
}
