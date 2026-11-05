export interface WordFilter {
    masked(text: string): string
}

export const NO_FILTER: WordFilter = {masked: (text) => text}

export const EVERY_WORD_MASKED: WordFilter = {masked: (text) => text.replace(/\S+/gu, maskedWord)}

export function wordFilterOf(entries: readonly string[]): WordFilter {
    return new ListFilter(entries)
}

type Word = {
    start: number
    end: number
    coreStart: number
    coreEnd: number
}

type Hit = {
    start: number
    end: number
    next: number
}

class ListFilter implements WordFilter {
    private readonly entries = new Set<string>()
    private readonly openings = new Set<string>()
    private readonly longest: number

    constructor(list: readonly string[]) {
        let longest = 0
        for (const line of list) {
            const words = line.toLowerCase().split(/\s+/u).filter((word) => word !== "")
            if (words.length === 0) continue
            this.entries.add(words.join(" "))
            for (let n = 1; n < words.length; n++) this.openings.add(words.slice(0, n).join(" "))
            longest = Math.max(longest, words.length)
        }
        this.longest = longest
    }

    masked(text: string): string {
        const words = wordsOf(text)
        let out = ""
        let at = 0
        for (let i = 0; i < words.length;) {
            const hit = this.hitAt(text, words, i)
            if (!hit) {
                i++
                continue
            }
            out += text.slice(at, hit.start) + text.slice(hit.start, hit.end).replace(/\S+/gu, maskedWord)
            at = hit.end
            i = hit.next
        }
        return at === 0 ? text : out + text.slice(at)
    }

    private hitAt(text: string, words: Word[], i: number): Hit | undefined {
        const starts = [...new Set([words[i].start, words[i].coreStart])]
        let hit: Hit | undefined
        for (let j = i; j < words.length && j - i < this.longest; j++) {
            const ends = [...new Set([words[j].end, words[j].coreEnd])]
            const found = starts.flatMap((start) => ends.map((end) => ({start, end})))
                .find(({start, end}) => start < end && this.entries.has(phraseOf(text, start, end)))
            if (found) hit = {...found, next: j + 1}
            if (!starts.some((start) => this.openings.has(phraseOf(text, start, words[j].end)))) break
        }
        return hit
    }
}

function wordsOf(text: string): Word[] {
    return [...text.matchAll(/\S+/gu)].map((match) => {
        const start = match.index
        const end = start + match[0].length
        const lead = /^[^\p{L}\p{N}\p{M}]*/u.exec(match[0])![0].length
        const trail = lead === match[0].length ? 0 : /[^\p{L}\p{N}\p{M}]*$/u.exec(match[0])![0].length
        return {start, end, coreStart: start + lead, coreEnd: end - trail}
    })
}

function phraseOf(text: string, start: number, end: number): string {
    return text.slice(start, end).replace(/\s+/gu, " ").toLowerCase()
}

function maskedWord(word: string): string {
    const letters = Array.from(word)
    if (letters.length === 1) return "*"
    if (letters.length === 2) return letters[0] + "*"
    return letters[0] + "*".repeat(letters.length - 2) + letters[letters.length - 1]
}
