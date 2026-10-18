import {Timestamp} from "@bufbuild/protobuf"
import {timeOf} from "./time.ts"

export function When({at}: {at: Timestamp | Date | undefined}) {
    if (at === undefined) return null
    const date = at instanceof Date ? at : at.toDate()
    return <time className="muted" dateTime={date.toISOString()}>{timeOf(date)}</time>
}
