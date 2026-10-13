import {CSSProperties} from "react"
import {ChatMessage, ChatQuote, GUEST_PREFIX} from "../../backends/chat.ts"
import {NameColor, PlayerLine} from "../../backends/player.ts"
import {hueOf} from "../../domain/authorColor.ts"

export type Painted = {color: NameColor | undefined, guest: boolean}

export function authorStyle({color, guest}: Painted): CSSProperties {
    const hue = guest ? undefined : hueOf(color)
    if (hue === undefined) return {"--author-hue": 0, "--author-chroma": 0} as CSSProperties
    return {"--author-hue": hue} as CSSProperties
}

// A quote sits inside its reply's line: with no chroma of its own it wears that author's grey.
export function quoteStyle(quote: ChatQuote): CSSProperties {
    return {
        "--author-chroma": 1,
        ...authorStyle({color: quote.authorColor, guest: quote.authorName.startsWith(GUEST_PREFIX)}),
    } as CSSProperties
}

export function authorOf(message: ChatMessage): PlayerLine {
    return {
        accountId: message.authorAccountId,
        name: message.authorName,
        countryCode: message.countryCode,
        guest: message.authorName.startsWith(GUEST_PREFIX),
        admin: message.authorAdmin,
        color: message.authorColor,
        streak: message.authorStreak,
        wornTitle: message.authorTitle,
    }
}
