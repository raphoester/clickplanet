import {useCallback, useEffect, useState} from "react"
import {putFile} from "../api.ts"
import {FlagsAdmin} from "../clients.ts"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {learnFlags, listedOf} from "../flags.ts"
import {Flag} from "../gen/grpc/flags/v1/flags_pb.ts"
import {countOf} from "../support/labels.ts"
import {useCall} from "../useCall.ts"
import {Put} from "../videos/actions.ts"
import {Order, soldOf, sortedOf} from "./actions.ts"
import {OpenPanel} from "./OpenPanel.tsx"
import {SoldCard} from "./SoldCard.tsx"

const ORDERS: {order: Order, label: string}[] = [{order: "end", label: "By end"}, {order: "name", label: "By name"}]

// The sold flags, opened by hand: a form to open one, then each with its days added, its rename, its close and its delete.
export function FlagsPage({flags, put = putFile}: {flags: FlagsAdmin, put?: Put}) {
    // The rest of the console names a flag by the same list: one opened here is named there at once.
    const list = useCallback(async (signal: AbortSignal) => {
        const answer = (await flags.listFlags({}, {signal})).flags
        learnFlags(listedOf(answer))
        return answer
    }, [flags])
    const [read, again] = useCall(list)
    // The list as last read stays while it is read again, so each card keeps what the server answered it.
    const [kept, setKept] = useState<Flag[]>()
    useEffect(() => {
        if (read.state === "done") setKept(read.value)
    }, [read])
    const [order, setOrder] = useState<Order>("end")
    // A code is never given again, so a flag deleted here stays out of the list whatever a read answers meanwhile.
    const [deleted, setDeleted] = useState<{codes: ReadonlySet<string>, last: Flag}>()

    const shown = read.state === "done" ? read.value : kept
    const sold = sortedOf(soldOf(shown ?? []).filter(each => deleted?.codes.has(each.flag.code) !== true), order)
    const open = sold.filter(each => each.closedAt === undefined).length
    const gone = (flag: Flag) => {
        setDeleted(was => ({codes: new Set([...was?.codes ?? [], flag.code]), last: flag}))
        again()
    }

    return (
        <div className="flags-page">
            <OpenPanel flags={flags} put={put} onOpened={again}/>
            <section className="flags-list" aria-label="Sold flags">
                <div className="toolbar">
                    <span className="muted">{shown === undefined ? "" : `${countOf(sold.length, "sold flag")}, ${open} open`}</span>
                    <div className="toolbar-end">
                        <div className="segmented" role="group" aria-label="Order">
                            {ORDERS.map(each => (
                                <button
                                    key={each.order}
                                    type="button"
                                    className="button"
                                    aria-pressed={order === each.order}
                                    onClick={() => setOrder(each.order)}
                                >
                                    {each.label}
                                </button>
                            ))}
                        </div>
                        <button type="button" className="button" disabled={read.state === "loading"} onClick={again}>
                            Reload
                        </button>
                    </div>
                </div>
                {read.state === "failed" && <FailureNotice failure={read} onRetry={again}/>}
                {deleted !== undefined && (
                    <p className="action-said" role="status">Deleted {deleted.last.name} ({deleted.last.code}) for good.</p>
                )}
                {shown === undefined && read.state === "loading" && <p className="muted">Loading…</p>}
                {shown !== undefined && sold.length === 0 && <p className="muted">No flag opened yet.</p>}
                {sold.length > 0 && (
                    <ol className="cards sold-grid">
                        {sold.map(each => (
                            <SoldCard key={each.flag.code} flags={flags} sold={each} onChanged={again} onDeleted={gone}/>
                        ))}
                    </ol>
                )}
            </section>
        </div>
    )
}
