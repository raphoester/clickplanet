import {Timestamp} from "@bufbuild/protobuf"
import {useState} from "react"
import {Clients} from "../clients.ts"
import {Person} from "../components/Person.tsx"
import {When} from "../components/When.tsx"
import {PlaceReport, ReportDecision} from "../gen/grpc/support/v1/admin_pb.ts"
import {Decided} from "../reports/actions.ts"
import {DecideForm} from "../reports/DecideForm.tsx"
import {DecisionLine} from "../reports/DecisionLine.tsx"
import {placeReasonLabel} from "../support/labels.ts"
import {beforeOf, decidePlace, DEFAULT_RINGS, mapLinkOf, MAX_RINGS} from "./actions.ts"
import {ClearPanel} from "./ClearPanel.tsx"
import {RestorePanel} from "./RestorePanel.tsx"

type Props = {
    report: PlaceReport
    clients: Clients
    onBack: () => void
}

export function DrawingDetail({report, clients, onBack}: Props) {
    const [decision, setDecision] = useState<ReportDecision | undefined>(report.decision)
    const [ringsValue, setRingsValue] = useState(String(DEFAULT_RINGS))
    const rings = ringsOf(ringsValue)

    const decided = (done: Decided) => setDecision(new ReportDecision({
        outcome: done.outcome,
        comment: done.comment,
        decidedAt: Timestamp.fromDate(done.decidedAt),
    }))

    return (
        <section className="detail" aria-label="Drawing report">
            <button type="button" className="button-small" onClick={onBack}>← Drawings</button>

            <article className="card">
                <div className="card-head">
                    <span className="badge badge-report">{placeReasonLabel(report.reason)}</span>
                    <When at={report.createdAt}/>
                </div>
                <dl className="people">
                    <Person label="By" name={report.reporterName} id={report.reporterAccountId}/>
                </dl>
                {report.text !== "" ? <p className="text">{report.text}</p> : <p className="muted">No details.</p>}
                <p>
                    Tile <span className="mono">{report.tileId}</span>{" · "}
                    <a href={mapLinkOf(report.tileId)} target="_blank" rel="noopener noreferrer">See it on the map ↗</a>
                </p>
                {decision !== undefined && <DecisionLine decision={decision}/>}
            </article>

            <section className="panel" aria-label="The area">
                <h2>The area</h2>
                <label className="picker">
                    <span className="muted">Rings around tile {report.tileId}</span>
                    <input
                        type="number"
                        min={1}
                        max={MAX_RINGS}
                        value={ringsValue}
                        onChange={event => setRingsValue(event.target.value)}
                    />
                </label>
            </section>

            <RestorePanel
                planet={clients.planet}
                player={clients.player}
                tileId={report.tileId}
                rings={rings}
                defaultBefore={beforeOf(report)}
            />
            <ClearPanel planet={clients.planet} tileId={report.tileId} rings={rings}/>

            <DecideForm
                decide={(outcome, comment) => decidePlace(clients.support, report.id, outcome, comment)}
                decided={decision !== undefined}
                onDecided={decided}
            />
        </section>
    )
}

// Whole rings from 1 to MAX_RINGS, or nothing while the field holds another value.
function ringsOf(value: string): number | undefined {
    const rings = Number(value)
    return Number.isInteger(rings) && rings >= 1 && rings <= MAX_RINGS ? rings : undefined
}
