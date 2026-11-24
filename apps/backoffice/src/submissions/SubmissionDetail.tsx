import {useMemo, useState} from "react"
import {Clients} from "../clients.ts"
import {Copyable} from "../components/AccountId.tsx"
import {Country} from "../components/Country.tsx"
import {Fact} from "../components/Fact.tsx"
import {FlagMark} from "../components/FlagMark.tsx"
import {When} from "../components/When.tsx"
import {countryLabel} from "../countries.ts"
import {globeLinkOf, kmWords, Place, placeWords} from "../flags/pin.ts"
import {KeptSubmission} from "../gen/grpc/flags/v1/admin_pb.ts"
import {Flag, SubmissionState} from "../gen/grpc/flags/v1/flags_pb.ts"
import {useOpenPlayer} from "../navigation.ts"
import {LetterPanel} from "../support/LetterPanel.tsx"
import {Call} from "../useCall.ts"
import {CapitalMap, Mark} from "./CapitalMap.tsx"
import {FlagFigure} from "./FlagFigure.tsx"
import {OpenSubmittedPanel} from "./OpenSubmittedPanel.tsx"
import {SubmissionVerdict} from "./SubmissionVerdict.tsx"
import {
    Around, aroundOf, founderOf, Ground, looksLikePhoto, NEAR_KM, nearOf, Other, reasonLabel, sameNameOf, stateLabel, toOpen,
    waiting,
} from "./submissions.ts"

type Props = {
    submission: KeptSubmission
    // Every submission read, and the catalog of flags: what it is compared with.
    submissions: KeptSubmission[]
    flags: Flag[]
    names: Map<string, string> | undefined
    // The board the world plays, to draw the capital on.
    ground: Call<Ground>
    clients: Clients
    onBack: () => void
    onOpen: (submissionId: string) => void
    onDecided: (decided: KeptSubmission, said: string) => void
    onOpened: (flag: Flag) => void
    onLettersChanged: () => void
}

// One submission, read top down as it is judged: the picture, the name, who sent it, its capital on a map and the
// capitals near it, the flags of the same name; then the verdict, or the open of an approved new flag. A waiting one's
// capital moves on the map before the verdict.
export function SubmissionDetail({
    submission, submissions, flags, names, ground, clients, onBack, onOpen, onDecided, onOpened, onLettersChanged,
}: Props) {
    const {pin, rejection} = submission
    const [moved, setMoved] = useState<Place>()
    const capital = moved ?? pin
    const capitals = ground.state === "done" ? ground.value.capitals : NO_CAPITALS
    const around = useMemo(() => aroundOf(submission, flags, submissions, capitals), [submission, flags, submissions, capitals])
    const marks = useMemo(() => around.map(markOf), [around])
    const near = useMemo(() => capital === undefined ? [] : nearOf(capital, around), [capital, around])
    const same = sameNameOf(submission, flags, submissions)
    const move = (place: Place) => setMoved(pin !== undefined && placeWords(place) === placeWords(pin) ? undefined : place)

    return (
        <section className="detail submission-detail" aria-label="Submission">
            <button type="button" className="button-small back" onClick={onBack}>← All submissions</button>
            <article className="card submission-card">
                <div className="card-head">
                    <span className={`badge ${badgeOf(submission.state)}`}>{stateLabel(submission.state)}</span>
                </div>
                {looksLikePhoto(submission) && (
                    <p className="photo-warning" role="alert">
                        {submission.otherColorsPercent}% of the picture is in colours past its 8 most common: most likely a
                        photo, not a flag.
                    </p>
                )}
                <FlagFigure picture={submission.picture} name={submission.name}/>
                <h3 className="submission-name">{submission.name}</h3>
                <dl className="facts">
                    <Fact label="Founder"><Founder accountId={submission.accountId} name={names?.get(submission.accountId) ?? ""}/></Fact>
                    <Fact label="Submitted"><When at={submission.submittedAt}/></Fact>
                    {!waiting(submission) && <Fact label={stateLabel(submission.state)}><When at={submission.decidedAt}/></Fact>}
                    {submission.picture !== undefined && (
                        <Fact label="Other colours">{submission.otherColorsPercent}% past the 8 most common</Fact>
                    )}
                    <Fact label="Capital">
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
            {pin !== undefined && capital !== undefined && (
                <section className="panel" aria-label="The capital">
                    <h2>The capital</h2>
                    {ground.state === "loading" && <p className="muted">Loading the map…</p>}
                    {ground.state === "failed" && <p className="muted">The map could not be drawn: {ground.detail}</p>}
                    {ground.state === "done" && (
                        <CapitalMap
                            places={ground.value.places}
                            capital={capital}
                            name={submission.name}
                            icon={<PictureMark submission={submission}/>}
                            marks={marks}
                            onMove={waiting(submission) ? move : undefined}
                            label={`The capital of ${submission.name} on a map of the world`}
                        />
                    )}
                    {moved !== undefined && (
                        <div className="capital-moved">
                            <span>
                                Moved to <strong>{placeWords(moved)}</strong> ·{" "}
                                <a href={globeLinkOf(moved)} target="_blank" rel="noreferrer">Open on the globe</a>
                            </span>
                            <button type="button" className="button-small" onClick={() => setMoved(undefined)}>
                                Back to the player's
                            </button>
                        </div>
                    )}
                    {moved === undefined && waiting(submission) && ground.state === "done" && (
                        <p className="hint">Drag the red mark, or tap the map, to move the capital.</p>
                    )}
                    <h3 className="near-title">Capitals near it</h3>
                    {near.length === 0
                        ? <p className="muted">No capital within {kmWords(NEAR_KM)}.</p>
                        : (
                            <ol className="near-list" aria-label="Capitals near it">
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
            {submission.accountId !== "" && (
                <LetterPanel
                    key={`letter-${submission.id}`}
                    support={clients.support}
                    submissionId={submission.id}
                    accountId={submission.accountId}
                    name={founderOf(submission.accountId, names)}
                    subject={`Your flag ${submission.name}`}
                    onChanged={onLettersChanged}
                />
            )}
            {waiting(submission) && (
                <SubmissionVerdict
                    key={submission.id}
                    flags={clients.flags}
                    submission={submission}
                    moved={moved}
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

const NO_CAPITALS: Ground["capitals"] = []

function keyOf(other: Other): string {
    switch (other.kind) {
        case "flag":
            return `flag-${other.flag.code}`
        case "submission":
            return `submission-${other.submission.id}`
        case "country":
            return `country-${other.country}`
    }
}

function markOf({other, place}: Around): Mark {
    const key = keyOf(other)
    switch (other.kind) {
        case "flag":
            return {key, place, kind: "flag", name: other.flag.name, icon: <FlagMark id={other.flag.code}/>}
        case "submission":
            return {key, place, kind: "submission", name: other.submission.name, icon: <PictureMark submission={other.submission}/>}
        case "country":
            return {key, place, kind: "country", name: countryLabel(other.country), icon: <FlagMark id={other.country}/>}
    }
}

// A submission's picture where a flag goes, before it is a flag.
function PictureMark({submission}: {submission: KeptSubmission}) {
    if (submission.picture === undefined) return null
    return <img className="flag flag-picture" src={submission.picture.url} alt="" aria-hidden="true"/>
}

// A flag the game ships with, a sold one, one of the game's countries, or another submission: one click to it.
function OtherLine({other, onOpen}: {other: Other, onOpen: (submissionId: string) => void}) {
    if (other.kind === "country") {
        return <span className="near-name"><Country id={other.country}/> <span className="muted">the game's own</span></span>
    }
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
