import {authorStyle} from "../chat/authorStyle.ts"
import {useWordFilter} from "../crazygames/wordFilter.ts"
import {truncate} from "../truncate.ts"
import {nameOf, paintOf} from "./nameplate.ts"
import {useNameplate} from "./nameplatesStore.ts"

// An account's name in its color, read by its id from the nameplates: nothing until it is read.
export default function PlayerName({accountId, className, max}: {accountId: string, className: string, max?: number}) {
    const nameplate = useNameplate(accountId)
    const name = useWordFilter().masked(nameOf(nameplate))

    return <span className={className} style={authorStyle(paintOf(nameplate))} title={name}>
        {max === undefined ? name : truncate(name, max)}
    </span>
}
