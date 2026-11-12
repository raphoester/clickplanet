import {Fragment, ReactNode} from "react"
import {Key, RichValues} from "./translator.ts"
import {translator} from "./page.ts"

// A line whose slots hold elements, as one sentence: rich("chat.knockedOut", {winner: <strong>…</strong>, …}).
export function rich(key: Key, values: RichValues<ReactNode>): ReactNode {
    return translator().parts<ReactNode>(key, values).map((part, i) => <Fragment key={i}>{part}</Fragment>)
}
