import {CSSProperties} from "react"
import {NameColor} from "../../backends/player.ts"
import {hueOf} from "../../domain/authorColor.ts"

export type Painted = {color: NameColor | undefined, guest: boolean}

export function authorStyle({color, guest}: Painted): CSSProperties {
    const hue = guest ? undefined : hueOf(color)
    if (hue === undefined) return {"--author-hue": 0, "--author-chroma": 0} as CSSProperties
    return {"--author-hue": hue} as CSSProperties
}

// A quote sits inside its reply's line: with no chroma of its own it wears that author's grey.
export function quoteStyle(painted: Painted): CSSProperties {
    return {"--author-chroma": 1, ...authorStyle(painted)} as CSSProperties
}
