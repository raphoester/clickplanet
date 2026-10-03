import {CSSProperties} from "react"
import {ChatMessage, GUEST_PREFIX} from "../../backends/chat.ts"
import {PlayerLine} from "../../backends/player.ts"
import {authorHue} from "../../domain/authorColor.ts"

export type Painted = Pick<PlayerLine, "name" | "color" | "guest">

export function authorStyle({name, color, guest}: Painted): CSSProperties {
    if (guest) return {"--author-hue": 0, "--author-chroma": 0} as CSSProperties
    return {"--author-hue": authorHue(name, color)} as CSSProperties
}

export function authorOf(message: ChatMessage): PlayerLine {
    return {
        name: message.authorName,
        countryCode: message.countryCode,
        guest: message.authorName.startsWith(GUEST_PREFIX),
        admin: message.authorAdmin,
        color: message.authorColor,
        streak: message.authorStreak,
        wornTitle: message.authorTitle,
    }
}
