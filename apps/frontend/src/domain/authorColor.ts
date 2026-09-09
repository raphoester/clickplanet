/**
 * A stable colour per chat author, so a run of messages can be told apart at a
 * glance without reading the small print of the name.
 *
 * The identity being coloured is the one the log displays: the name *and* the
 * `author_tag` the server stamps. Two people typing the same name get two
 * colours, and someone renaming themselves gets a new one — which is what the
 * reader sees happen too.
 *
 * Only the hue is derived. Saturation and lightness are fixed in the CSS, so
 * every author lands at the same legibility against the dark panel and no hash
 * can produce an unreadable one.
 */
export function authorHue(authorName: string, authorTag: string): number {
    return hash(`${authorName}#${authorTag}`) % 360
}

const FNV_OFFSET = 0x811c9dc5
const FNV_PRIME = 0x01000193

function hash(key: string): number {
    let value = FNV_OFFSET
    for (const rune of key) {
        value ^= rune.codePointAt(0)!
        value = Math.imul(value, FNV_PRIME)
    }
    return value >>> 0
}
