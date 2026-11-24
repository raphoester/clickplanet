import {useMemo} from "react"
import {Places} from "../boards/places.ts"
import {Clients} from "../clients.ts"
import {Copyable} from "../components/AccountId.tsx"
import {Fact} from "../components/Fact.tsx"
import {FlagMark} from "../components/FlagMark.tsx"
import {When} from "../components/When.tsx"
import {globeLinkOf, kmWords, placeWords} from "../flags/pin.ts"
import {KeptSubmission} from "../gen/grpc/flags/v1/admin_pb.ts"
import {Flag, SubmissionState} from "../gen/grpc/flags/v1/flags_pb.ts"
import {useOpenPlayer} from "../navigation.ts"
import {Call} from "../useCall.ts"
import {FlagFigure} from "./FlagFigure.tsx"
import {OpenSubmittedPanel} from "./OpenSubmittedPanel.tsx"
import {PinMap} from "./PinMap.tsx"
import {SubmissionVerdict} from "./SubmissionVerdict.tsx"
import {
    kindOf, looksLikePhoto, NEAR_KM, nearOf, Other, reasonLabel, sameNameOf, stateLabel, toOpen, waiting,
} from "./submissions.ts"

type Props = {
    submission: KeptSubmission
    // Every submission read, and the catalog of flags: what it is compared with.
    submissions: KeptSubmission[]
    flags: Flag[]
    names: Map<string, string> | undefined
    // The places of the board the world plays, to draw the land around the pin.
    places: Call<Places>
    clients: Clients
    onBack: () => void
    onOpen: (submissionId: string) => void
    onDecided: (decided: KeptSubmission, said: string) => void
    onOpened: (flag: Flag) => void
}

// One submission, read top down as it is judged: the picture, the name, who sent it, the pin and the flags near it,
// the flags of the same name; then the verdict, or the open of an approved new flag.
export function SubmissionDetail({submission, submissions, flags, names, places, clients, onBack, onOpen, onDecided, onOpened}: Props) {
    const changed = submission.change ? flags.find(flag => flag.code === submission.flag) : undefined
    const near = useMemo(() => nearOf(submission, flags, submissions), [submission, flags, submissions])
    const marked = useMemo(() => near.map(each => ({place: each.place, name: nameOf(each.other)})), [near])
    const same = sameNameOf(submission, flags, submissions)
    const {pin, rejection} = submission

    return (
        <section className="detail submission-detail" aria-label="Submission">
            <button type="button" className="button-small back" onClick={onBack}>← All submissions</button>
            <article className="card submission-card">
                <div className="card-head">
                    <span className={`badge ${badgeOf(submission.state)}`}>{stateLabel(submission.state)}</span>
                    <span className="muted">{kindOf(submission, flags)}</span>
                </div>
                {looksLikePhoto(submission) && (
                    <p className="photo-warning" role="alert">
                        {submission.otherColorsPercent}% of the picture is in colours past its 8 most common: most likely a
                        photo, not a flag.
                    </p>
                )}
                <div className="submission-figures">
                    {submission.change && (
                        <FlagFigure
                            picture={changed?.sold?.picture}
                            name={changed?.name ?? submission.flag}
                            caption={`Now: ${changed?.name ?? submission.flag}`}
                        />
                    )}
                    <FlagFigure
                        picture={submission.picture}
                        name={submission.name}
                        caption={submission.change ? `Asked: ${submission.name}` : undefined}
                    />
                </div>
                <h3 className="submission-name">{submission.name}</h3>
                <dl className="facts">
                    <Fact label="Founder"><Founder accountId={submission.accountId} name={names?.get(submission.accountId) ?? ""}/></Fact>
                    <Fact label="Submitted"><When at={submission.submittedAt}/></Fact>
                    {!waiting(submission) && <Fact label={stateLabel(submission.state)}><When at={submission.decidedAt}/></Fact>}
                    {submission.picture !== undefined && (
                        <Fact label="Other colours">{submission.otherColorsPercent}% past the 8 most common</Fact>
                    )}
                    <Fact label="Pin">
                        {pin === undefined
                            ? <span className="muted">none</span>
                            : <>{placeWords(pin)} · <a href={globeLinkOf(pin)} target="_blank" rel="noreferrer">Open on the globe</a></>}
                    </Fact>
                    {submission.flag !== "" && <Fact label="Flag"><FlagMark id={submission.flag}/> <Copyable text={submission.flag}/></Fact>}
                    {rejection !== undefined && (
                        <>
                            <Fact label="Reason">{reasonLabel(rejection.reason)}</Fact>
                            {rejection.note !== "" && <Fact label="Note"><span className="text">{rejection.note}</span></Fact>}
                        </>
                    )}
                </dl>
            </article>
            {pin !== undefined && (
                <section className="panel" aria-label="The pin">
                    <h2>The pin</h2>
                    {places.state === "loading" && <p className="muted">Loading the map…</p>}
                    {places.state === "failed" && <p className="muted">The map could not be drawn: {places.detail}</p>}
                    {places.state === "done" && (
                        <PinMap
                            places={places.value}
                            pin={pin}
                            others={marked}
                            label={`The land around the pin of ${submission.name}, north up`}
                        />
                    )}
                    <h3 className="near-title">Near it</h3>
                    {near.length === 0
                        ? <p className="muted">No flag within {kmWords(NEAR_KM)}.</p>
                        : (
                            <ol className="near-list" aria-label="Flags near it">
                                {near.map(each => (
                                    <li key={keyOf(each.other)}>
                                        <OtherLine other={each.other} onOpen={onOpen}/>
                                        <span className="near-km">{kmWords(each.km)}</span>
                                    </li>
                                ))}
                            </ol>
                        )}
                </section>
            )}
            {same.length > 0 && (
                <section className="panel" aria-label="The same name">
                    <h2>The same name</h2>
                    <ul className="near-list">
                        {same.map(other => <li key={keyOf(other)}><OtherLine other={other} onOpen={onOpen}/></li>)}
                    </ul>
                </section>
            )}
            {waiting(submission) && (
                <SubmissionVerdict
                    key={submission.id}
                    flags={clients.flags}
                    submission={submission}
                    changes={submission.change ? changed?.name ?? submission.flag : undefined}
                    onDecided={onDecided}
                />
            )}
            {toOpen(submission) && <OpenSubmittedPanel flags={clients.flags} submission={submission} onOpened={onOpened}/>}
        </section>
    )
}

function badgeOf(state: SubmissionState): string {
    switch (state) {
        case SubmissionState.WAITING:
            return "badge-report"
        case SubmissionState.APPROVED:
        case SubmissionState.OPENED:
            return "badge-acted"
        case SubmissionState.REJECTED:
        case SubmissionState.UNSPECIFIED:
            return "badge-dismissed"
    }
}

function nameOf(other: Other): string {
    return other.kind === "flag" ? other.flag.name : other.submission.name
}

function keyOf(other: Other): string {
    return other.kind === "flag" ? `flag-${other.flag.code}` : `submission-${other.submission.id}`
}

// A flag the game ships with, a sold one, or another submission: one click to it.
function OtherLine({other, onOpen}: {other: Other, onOpen: (submissionId: string) => void}) {
    if (other.kind === "flag") {
        const {flag} = other
        const what = flag.sold === undefined ? "the game's own" : flag.closed ? "sold, closed" : "sold"
        return (
            <span className="near-name">
                <FlagMark id={flag.code}/> <span className="name">{flag.name}</span> <span className="muted">{what}</span>
            </span>
        )
    }
    const {submission} = other
    return (
        <span className="near-name">
            <button type="button" className="line-author" onClick={() => onOpen(submission.id)}>{submission.name}</button>
            {" "}
            <span className="muted">
                submitted, {stateLabel(submission.state).toLowerCase()}
                {submission.rejection !== undefined && `: ${reasonLabel(submission.rejection.reason)}`}
            </span>
        </span>
    )
}

// The name opens the player page, with every tool on the account. With no player of that id, the id alone.
function Founder({accountId, name}: {accountId: string, name: string}) {
    const openPlayer = useOpenPlayer()
    if (name === "") return <Copyable text={accountId}/>
    return (
        <span className="painter">
            {openPlayer === undefined
                ? <span className="name">{name}</span>
                : (
                    <button
                        type="button"
                        className="line-author"
                        title="Open on the player page"
                        onClick={() => openPlayer({caller: {accountId}, name})}
                    >
                        {name}
                    </button>
                )}
            <Copyable text={accountId} short/>
        </span>
    )
}
