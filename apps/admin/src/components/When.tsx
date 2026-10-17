import {Timestamp} from "@bufbuild/protobuf"

const FORMAT = new Intl.DateTimeFormat(undefined, {dateStyle: "medium", timeStyle: "short"})

export function When({at}: {at: Timestamp | undefined}) {
    if (at === undefined) return null
    const date = at.toDate()
    return <time className="muted" dateTime={date.toISOString()}>{FORMAT.format(date)}</time>
}
