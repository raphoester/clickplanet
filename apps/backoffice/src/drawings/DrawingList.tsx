import {useCallback} from "react"
import {SupportAdmin} from "../clients.ts"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {When} from "../components/When.tsx"
import {PlaceReport} from "../gen/grpc/support/v1/admin_pb.ts"
import {DecisionLine} from "../reports/DecisionLine.tsx"
import {countOf, placeReasonLabel} from "../support/labels.ts"
import {useCall} from "../useCall.ts"
import {listPlaceReports} from "./actions.ts"

type Props = {
    support: SupportAdmin
    openOnly: boolean
    onOpenOnly: (openOnly: boolean) => void
    onOpen: (report: PlaceReport) => void
}

export function DrawingList({support, openOnly, onOpenOnly, onOpen}: Props) {
    const list = useCallback(
        (signal: AbortSignal) => listPlaceReports(support, openOnly, signal),
        [support, openOnly],
    )
    const [reports, again] = useCall(list)

    return (
        <section aria-label="Drawings">
            <div className="toolbar">
                <div className="segmented" role="group" aria-label="Which reports">
                    <button type="button" aria-pressed={openOnly} onClick={() => onOpenOnly(true)}>Open</button>
                    <button type="button" aria-pressed={!openOnly} onClick={() => onOpenOnly(false)}>All</button>
                </div>
                <button type="button" className="button" onClick={again}>Reload</button>
            </div>

            {reports.state === "loading" && <p className="muted">Loading…</p>}
            {reports.state === "failed" && <FailureNotice failure={reports} onRetry={again}/>}
            {reports.state === "done" && (
                <>
                    <p className="muted">{countOf(reports.value.length, openOnly ? "open report" : "report")}</p>
                    <ol className="cards">
                        {reports.value.map(report => (
                            <DrawingCard key={report.id} report={report} onOpen={() => onOpen(report)}/>
                        ))}
                    </ol>
                </>
            )}
        </section>
    )
}

function DrawingCard({report, onOpen}: {report: PlaceReport, onOpen: () => void}) {
    return (
        <li className="card">
            <div className="card-head">
                <span className="badge badge-report">{placeReasonLabel(report.reason)}</span>
                <When at={report.createdAt}/>
            </div>
            <p className="muted">
                Tile <span className="mono">{report.tileId}</span>, reported by{" "}
                {report.reporterName === "" ? "an unnamed player" : <span className="name">{report.reporterName}</span>}
            </p>
            {report.text !== "" && <p className="text">{report.text}</p>}
            <div className="card-foot">
                {report.decision !== undefined ? <DecisionLine decision={report.decision}/> : <span/>}
                <button type="button" className="button" onClick={onOpen}>Open</button>
            </div>
        </li>
    )
}
