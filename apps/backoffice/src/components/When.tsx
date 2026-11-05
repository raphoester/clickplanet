import {Timestamp} from "@bufbuild/protobuf"
import {agoOf, timeOf} from "./time.ts"

export function When({at}: {at: Timestamp | Date | undefined}) {
    if (at === undefined) return null
    const date = at instanceof Date ? at : at.toDate()
    return <time className="muted" dateTime={date.toISOString()}>{timeOf(date)}</time>
}

// How long ago, the time itself on hover: for a list scanned by age.
export function Ago({at}: {at: Timestamp | Date | undefined}) {
    if (at === undefined) return null
    const date = at instanceof Date ? at : at.toDate()
    return <time className="muted" dateTime={date.toISOString()} title={timeOf(date)}>{agoOf(date)}</time>
}
