import {useCallback, useEffect, useState} from "react"
import {SupportAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {Letter} from "../gen/grpc/support/v1/admin_pb.ts"
import {useCall} from "../useCall.ts"
import {MAX_ANSWER_LENGTH} from "./actions.ts"
import {
    closeLetter, latestLettersFirst, letterRefusalOf, MAX_SUBJECT, threadOf, waitsOf, writeLetter, Writing,
} from "./letters.ts"
import {LineHead} from "./NoteDetail.tsx"

type Props = {
    support: SupportAdmin
    accountId: string
    // The player as the page names it.
    name: string
    // A new letter's subject at first.
    subject: string
    // A letter sent or closed: what waits for the operator changed.
    onChanged: () => void
}

// The letters to one player, each a thread of the mails and the replies, one more mail in any, and a new one.
export function LettersPanel({support, accountId, name, subject, onChanged}: Props) {
    const read = useCallback(
        async (signal: AbortSignal) => (await support.listLetters({accountId, limit: 100}, {signal})).letters,
        [support, accountId],
    )
    const [call, again] = useCall(read)
    const [letters, setLetters] = useState<Letter[]>()
    useEffect(() => {
        if (call.state === "done") setLetters(call.value)
    }, [call])

    const kept = (letter: Letter) => {
        setLetters(was => latestLettersFirst([letter, ...(was ?? []).filter(each => each.id !== letter.id)]))
        onChanged()
    }
    const address = letters?.find(letter => letter.address !== "")?.address

    return (
        <section className="panel letters" aria-label={`Write to ${name}`}>
            <h2>Write to {name}</h2>
            {call.state === "failed" && <FailureNotice failure={call} onRetry={again}/>}
            {letters === undefined && call.state === "loading" && <p className="muted">Loading the letters…</p>}
            {letters?.map(letter => <LetterThread key={letter.id} letter={letter} name={name} support={support} onKept={kept}/>)}
            {letters !== undefined && (
                <NewLetter
                    support={support}
                    accountId={accountId}
                    to={address ?? `${name}'s verified address`}
                    subject={subject}
                    onKept={kept}
                />
            )}
        </section>
    )
}

function LetterThread({letter, name, support, onKept}: {
    letter: Letter, name: string, support: SupportAdmin, onKept: (letter: Letter) => void,
}) {
    const [text, setText] = useState("")
    const waits = waitsOf(letter)

    return (
        <article className="letter" aria-label={`Letter: ${letter.subject}`}>
            <div className="card-head">
                <h3 className="letter-subject">{letter.subject}</h3>
                {waits && <span className="badge badge-replied">Replied</span>}
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
            {letter.address !== "" && (
                <>
                    <label className="field">
                        <span className="muted">Write again</span>
                        <textarea value={text} maxLength={MAX_ANSWER_LENGTH} rows={4} onChange={event => setText(event.target.value)}/>
                    </label>
                    <div className="verdict-buttons">
                        <ConfirmAction
                            label="Send"
                            disabled={text.trim() === ""}
                            question={`Send this to ${letter.address}?`}
                            run={() => writeLetter(support, {letterId: letter.id}, text)}
                            said={sent => `Sent to ${sent.address}.`}
                            refusal={err => letterRefusalOf("write", err)}
                            onDone={sent => {
                                setText("")
                                onKept(sent)
                            }}
                        />
                        {waits && (
                            <ConfirmAction
                                label="Close"
                                question="Close this letter with no further mail? It waits for nothing until the player writes again."
                                run={() => closeLetter(support, letter)}
                                said={() => "Closed."}
                                refusal={err => letterRefusalOf("close", err)}
                                onDone={onKept}
                            />
                        )}
                    </div>
                </>
            )}
        </article>
    )
}

function NewLetter({support, accountId, to, subject: first, onKept}: {
    support: SupportAdmin, accountId: string, to: string, subject: string, onKept: (letter: Letter) => void,
}) {
    const [subject, setSubject] = useState(first)
    const [text, setText] = useState("")
    const writing: Writing = {accountId, subject}
    const subjectLength = [...subject.trim()].length

    return (
        <div className="letter letter-new">
            <h3 className="letter-subject">New letter</h3>
            <label className="field">
                <span className="muted">Subject · {subjectLength}/{MAX_SUBJECT}</span>
                <input value={subject} onChange={event => setSubject(event.target.value)}/>
            </label>
            {subjectLength > MAX_SUBJECT && <p className="action-refused" role="alert">A subject of {MAX_SUBJECT} characters at most.</p>}
            <label className="field">
                <span className="muted">Text</span>
                <textarea value={text} maxLength={MAX_ANSWER_LENGTH} rows={5} onChange={event => setText(event.target.value)}/>
            </label>
            <ConfirmAction
                label="Send"
                disabled={text.trim() === "" || subjectLength === 0 || subjectLength > MAX_SUBJECT}
                question={`Send this to ${to}?`}
                run={() => writeLetter(support, writing, text)}
                said={sent => `Sent to ${sent.address}.`}
                refusal={err => letterRefusalOf("write", err)}
                onDone={sent => {
                    setText("")
                    onKept(sent)
                }}
            />
        </div>
    )
}
