import {Code, ConnectError} from "@connectrpc/connect"
import {useCallback} from "react"
import {Conversation} from "../chat/Conversation.tsx"
import {theirTitleOf} from "../chat/theirConversations.ts"
import {TheirMessages} from "../chat/TheirMessages.tsx"
import {Clients, CLIENTS} from "../clients.ts"
import {Copyable} from "../components/AccountId.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {Fold} from "../components/Fold.tsx"
import {Ago, When} from "../components/When.tsx"
import {GetMessageResponse, KeptMessage} from "../gen/grpc/chat/v1/admin_pb.ts"
import {Report} from "../gen/grpc/support/v1/admin_pb.ts"
import {useOpenPlayer} from "../navigation.ts"
import {mutingOf, readingOf} from "../players/actions.ts"
import {whoOf} from "../players/caller.ts"
import {Inspection} from "../players/InspectPanel.tsx"
import {Standing} from "../players/Standing.tsx"
import {outcomeLabel, reasonLabel} from "../support/labels.ts"
import {Call, useCall} from "../useCall.ts"
import {MESSAGE_GONE} from "./actions.ts"
import {DecisionLine} from "./DecisionLine.tsx"
import {Closed} from "./verdict.ts"
import {VerdictPanel} from "./VerdictPanel.tsx"

type Read = {kept: true, message: GetMessageResponse} | {kept: false} | {none: true}

type Props = {
    report: Report
    // Every other report on the same reported account, open or closed.
    related?: Report[]
    clients?: Clients
    onBack: () => void
    onOpen?: (reportId: string) => void
    onClosed?: (closed: Closed) => void
}

// One report, read top down as it is judged: what was said, who said it and where it stands, the verdict, then the
// chat around it and everything else the reported account said.
export function ReportDetail({report, related = [], clients = CLIENTS, onBack, onOpen, onClosed}: Props) {
    const {chat, antibot} = clients
    const messageId = report.messageId
    const accountId = report.reportedAccountId
    const parties = {
        reporter: {accountId: report.reporterAccountId, name: report.reporterName},
        reported: {accountId, name: report.reportedName},
    }
    const who = whoOf({accountId}, report.reportedName)

    const readMessage = useCallback(
        (signal: AbortSignal): Promise<Read> => messageId === "" ? Promise.resolve({none: true}) : chat.getMessage({messageId}, {signal}).then(
            message => ({kept: true, message}),
            (err: unknown) => {
                if (ConnectError.from(err).code === Code.NotFound) return {kept: false}
                throw err
            },
        ),
        [chat, messageId],
    )
    const [message, readAgain] = useCall(readMessage)
    const readStanding = useCallback(
        (signal: AbortSignal) => readingOf(antibot, {accountId}, signal),
        [antibot, accountId],
    )
    const [standing] = useCall(readStanding)
    const readMuting = useCallback(
        (signal: AbortSignal) => accountId === "" ? Promise.resolve(undefined) : mutingOf(chat, accountId, signal),
        [chat, accountId],
    )
    const [muting] = useCall(readMuting)
    const kept = message.state === "done" && "kept" in message.value && message.value.kept ? message.value.message : undefined
    const others = related.filter(other => other.decision === undefined)

    return (
        <section className="detail report-detail" aria-label="Report">
            <button type="button" className="button-small back" onClick={onBack}>← All reports</button>
            <article className="card report-head">
                <div className="card-head">
                    <span className="badge badge-report">{reasonLabel(report.reason)}</span>
                    <When at={report.createdAt}/>
                </div>
                <Said message={message} again={readAgain}/>
                {report.text !== "" && (
                    <p className="report-words"><span className="muted">{nameOf(report.reporterName)} wrote:</span> {report.text}</p>
                )}
                <dl className="people">
                    <Party label="Reported" name={report.reportedName} id={accountId}/>
                    <Party label="By" name={report.reporterName} id={report.reporterAccountId}/>
                </dl>
                {accountId !== "" && <div className="standing-line"><Standing reading={standing} muting={muting}/></div>}
                {related.length > 0 && <Related reports={related} onOpen={onOpen}/>}
                {report.decision !== undefined && <DecisionLine decision={report.decision}/>}
            </article>
            <VerdictPanel
                key={report.id}
                clients={clients}
                report={report}
                who={who}
                removable={kept !== undefined && kept.removedAt === undefined}
                others={others}
                onClosed={closed => onClosed?.(closed)}
            />
            {kept !== undefined && (
                <section className="panel" aria-label="The message">
                    <h2>The chat around it</h2>
                    <Conversation
                        chat={chat}
                        anchor={keptOf(kept, report)}
                        removedAt={kept.removedAt?.toDate()}
                        parties={parties}
                        mark="Reported message"
                    />
                </section>
            )}
            {accountId !== "" && (
                <Fold label="Their messages" title={theirTitleOf(report.reportedName)} open={messageId === ""}>
                    <TheirMessages chat={chat} accountId={accountId} parties={parties}/>
                </Fold>
            )}
            {accountId !== "" && standing.state === "done" && standing.value.on && (
                <Fold label="Antibot" title="What the antibot reads">
                    <Inspection value={standing.value.value} caller={{accountId}}/>
                </Fold>
            )}
        </section>
    )
}

// The reported line itself, quoted large: most reports are judged on it alone.
function Said({message, again}: {message: Call<Read>, again: () => void}) {
    if (message.state === "loading") return <p className="muted">Loading the message…</p>
    if (message.state === "failed") return <FailureNotice failure={message} onRetry={again}/>
    if ("none" in message.value) return null
    if (!message.value.kept) return <p className="muted">{MESSAGE_GONE}</p>
    const removed = message.value.message.removedAt
    return (
        <blockquote className="report-quote">
            <p className={removed === undefined ? "line-text" : "line-text line-removed"}>{message.value.message.text}</p>
            {removed !== undefined && <p className="hint">Removed <When at={removed}/>.</p>}
        </blockquote>
    )
}

function Party({label, name, id}: {label: string, name: string, id: string}) {
    const openPlayer = useOpenPlayer()
    return (
        <div>
            <dt>{label}</dt>
            <dd>
                {openPlayer === undefined || id === ""
                    ? <span className={name === "" ? "muted" : "name"}>{nameOf(name)}</span>
                    : (
                        <button
                            type="button"
                            className="line-author"
                            title="Open on the player page"
                            onClick={() => openPlayer({caller: {accountId: id}, name})}
                        >
                            {nameOf(name)}
                        </button>
                    )}
                {id === "" ? <span className="muted">no account</span> : <Copyable text={id} short/>}
            </dd>
        </div>
    )
}

// The account's other reports, newest first: one click to each.
function Related({reports, onOpen}: {reports: Report[], onOpen?: (reportId: string) => void}) {
    return (
        <div className="related">
            <span className="muted">Reported {reports.length === 1 ? "once more" : `${reports.length} more times`}:</span>
            <ul className="chips">
                {reports.map(other => (
                    <li key={other.id}>
                        <button type="button" className="chip" disabled={onOpen === undefined} onClick={() => onOpen?.(other.id)}>
                            {reasonLabel(other.reason)} · <Ago at={other.createdAt}/> ·{" "}
                            {other.decision === undefined ? "open" : outcomeLabel(other.decision.outcome)}
                        </button>
                    </li>
                ))}
            </ul>
        </div>
    )
}

function nameOf(name: string): string {
    return name === "" ? "no name" : name
}

// GetMessage names no author: the report does, for the account it names.
function keptOf(message: GetMessageResponse, report: Report): KeptMessage {
    const name = [
        {accountId: report.reportedAccountId, name: report.reportedName},
        {accountId: report.reporterAccountId, name: report.reporterName},
    ].find(each => each.accountId === message.accountId)?.name
    return new KeptMessage({
        messageId: message.messageId,
        accountId: message.accountId,
        authorName: message.accountId === "" ? "" : name ?? "",
        text: message.text,
        sentAt: message.sentAt,
        countryId: message.countryId,
        removedAt: message.removedAt,
    })
}
