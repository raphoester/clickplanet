import {ChatMessage, GUEST_PREFIX} from "../backends/chat.ts"

export type TextPart =
    | {kind: "text", text: string}
    | {kind: "link", text: string, href: string}

const URL_PATTERN = /https?:\/\/[^\s<>"]+/gi

const TRAILING_PUNCTUATION = /[.,;:!?'"]$/

export function messageParts(message: ChatMessage): TextPart[] {
    if (message.authorName.startsWith(GUEST_PREFIX)) return [{kind: "text", text: message.text}]
    return textParts(message.text)
}

export function textParts(text: string): TextPart[] {
    const parts: TextPart[] = []
    let end = 0

    for (const match of text.matchAll(URL_PATTERN)) {
        const written = withoutTrailingPunctuation(match[0])
        const href = hrefOf(written)
        if (href === undefined) continue

        if (match.index > end) parts.push({kind: "text", text: text.slice(end, match.index)})
        parts.push({kind: "link", text: written, href})
        end = match.index + written.length
    }

    if (end < text.length) parts.push({kind: "text", text: text.slice(end)})
    return parts
}

function withoutTrailingPunctuation(candidate: string): string {
    let written = candidate
    while (TRAILING_PUNCTUATION.test(written) || closesUnopened(written)) written = written.slice(0, -1)
    return written
}

function closesUnopened(written: string): boolean {
    return written.endsWith(")") && written.split(")").length > written.split("(").length
}

// "https://clickplanet.lol@evil.example" reads as this site and goes to the other one.
function hrefOf(written: string): string | undefined {
    let url: URL
    try {
        url = new URL(written)
    } catch {
        return undefined
    }
    if (url.username !== "" || url.password !== "") return undefined
    return url.href
}
