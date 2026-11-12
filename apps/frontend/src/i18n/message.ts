// A subset of ICU MessageFormat: {slot}, {n, plural, …}, {n, selectordinal, …}, {v, select, …}, # in a plural, and
// <tag>…</tag> around words a caller wraps. No escapes: a brace is always syntax, an apostrophe always a letter.
export type Node =
    | {kind: "text", text: string}
    | {kind: "slot", name: string}
    | {kind: "count"}
    | {kind: "plural", name: string, ordinal: boolean, branches: ReadonlyMap<string, Node[]>}
    | {kind: "select", name: string, branches: ReadonlyMap<string, Node[]>}
    | {kind: "tag", name: string, children: Node[]}

export class MessageError extends Error {
}

const NAME = /[A-Za-z_][A-Za-z0-9_]*/y
const SELECTOR = /=\d+|[A-Za-z_][A-Za-z0-9_]*/y
const TAG_OPEN = /<([A-Za-z][A-Za-z0-9-]*)>/y
const BLANK = /\s*/y

export function parseMessage(text: string): Node[] {
    const parser = new Parser(text)
    const nodes = parser.nodes({inCount: false})
    if (parser.at < text.length) throw parser.error(`unexpected "${text[parser.at]}"`)
    return nodes
}

class Parser {
    at = 0

    constructor(private readonly text: string) {
    }

    nodes(context: {inCount: boolean, closing?: string}): Node[] {
        const nodes: Node[] = []
        let text = ""
        const flush = () => {
            if (text) nodes.push({kind: "text", text})
            text = ""
        }
        while (this.at < this.text.length) {
            const char = this.text[this.at]
            if (char === "}") break
            if (context.closing !== undefined && this.text.startsWith(`</${context.closing}>`, this.at)) break
            if (char === "{") {
                flush()
                nodes.push(this.argument())
                continue
            }
            if (char === "#" && context.inCount) {
                flush()
                nodes.push({kind: "count"})
                this.at++
                continue
            }
            const tag = this.match(TAG_OPEN)
            if (tag) {
                flush()
                const name = tag[1]
                const children = this.nodes({inCount: context.inCount, closing: name})
                if (!this.text.startsWith(`</${name}>`, this.at)) throw this.error(`<${name}> is not closed`)
                this.at += name.length + 3
                nodes.push({kind: "tag", name, children})
                continue
            }
            if (this.text.startsWith("</", this.at)) throw this.error("a closing tag with no opening")
            text += char
            this.at++
        }
        flush()
        return nodes
    }

    private argument(): Node {
        this.at++
        this.skipBlank()
        const name = this.expect(NAME, "a slot's name")[0]
        this.skipBlank()
        if (this.text[this.at] === "}") {
            this.at++
            return {kind: "slot", name}
        }
        this.expectChar(",")
        this.skipBlank()
        const type = this.expect(NAME, "plural, selectordinal or select")[0]
        this.skipBlank()
        this.expectChar(",")
        const branches = new Map<string, Node[]>()
        const counted = type === "plural" || type === "selectordinal"
        if (!counted && type !== "select") throw this.error(`"${type}" is not plural, selectordinal or select`)
        for (; ;) {
            this.skipBlank()
            if (this.text[this.at] === "}") break
            const selector = this.expect(SELECTOR, "a branch's selector")[0]
            if (branches.has(selector)) throw this.error(`the branch "${selector}" is there twice`)
            this.skipBlank()
            this.expectChar("{")
            branches.set(selector, this.nodes({inCount: counted}))
            this.expectChar("}")
        }
        this.at++
        if (!branches.has("other")) throw this.error(`{${name}} has no "other" branch`)
        return counted
            ? {kind: "plural", name, ordinal: type === "selectordinal", branches}
            : {kind: "select", name, branches}
    }

    private match(pattern: RegExp): RegExpExecArray | null {
        pattern.lastIndex = this.at
        const found = pattern.exec(this.text)
        if (found) this.at = pattern.lastIndex
        return found
    }

    private expect(pattern: RegExp, what: string): RegExpExecArray {
        const found = this.match(pattern)
        if (!found) throw this.error(`${what} expected`)
        return found
    }

    private expectChar(char: string) {
        if (this.text[this.at] !== char) throw this.error(`"${char}" expected`)
        this.at++
    }

    private skipBlank() {
        this.match(BLANK)
    }

    error(problem: string): MessageError {
        return new MessageError(`${problem} at ${this.at} in "${this.text}"`)
    }
}

// What a message asks of its caller: each name, and whether it is a count, a choice or words wrapped in a tag.
export type Slot = "value" | "count" | "ordinal" | "choice" | "tag"

export function slotsOf(nodes: readonly Node[], into = new Map<string, Slot>()): Map<string, Slot> {
    for (const node of nodes) {
        switch (node.kind) {
            case "text":
            case "count":
                break
            case "slot":
                if (!into.has(node.name)) into.set(node.name, "value")
                break
            case "plural":
                into.set(node.name, node.ordinal ? "ordinal" : "count")
                for (const branch of node.branches.values()) slotsOf(branch, into)
                break
            case "select":
                into.set(node.name, "choice")
                for (const branch of node.branches.values()) slotsOf(branch, into)
                break
            case "tag":
                into.set(node.name, "tag")
                slotsOf(node.children, into)
                break
        }
    }
    return into
}

// Every plural in a message, with the branches it has: a language's file must give each the categories it needs.
export function pluralsOf(nodes: readonly Node[], into: {name: string, ordinal: boolean, selectors: string[]}[] = []) {
    for (const node of nodes) {
        switch (node.kind) {
            case "text":
            case "count":
            case "slot":
                break
            case "plural":
                into.push({name: node.name, ordinal: node.ordinal, selectors: [...node.branches.keys()]})
                for (const branch of node.branches.values()) pluralsOf(branch, into)
                break
            case "select":
                for (const branch of node.branches.values()) pluralsOf(branch, into)
                break
            case "tag":
                pluralsOf(node.children, into)
                break
        }
    }
    return into
}

export type Wrap<T> = (children: (string | T)[]) => T

export type Value<T> = string | number | T | Wrap<T>

export type Speech = {
    number(n: number): string
    category(n: number, ordinal: boolean): Intl.LDMLPluralRule
}

export class MissingValueError extends Error {
}

export function formatMessage<T>(nodes: readonly Node[], values: Readonly<Record<string, Value<T>>>, speech: Speech): (string | T)[] {
    const valueOf = (name: string): Value<T> => {
        if (!(name in values)) throw new MissingValueError(`no value for {${name}}`)
        return values[name]
    }
    const countOf = (name: string): number => {
        const value = valueOf(name)
        if (typeof value !== "number") throw new MissingValueError(`{${name}} is counted and needs a number`)
        return value
    }
    const walk = (list: readonly Node[], count: number | undefined, parts: (string | T)[]): (string | T)[] => {
        const push = (part: string | T) => {
            const last = parts.length - 1
            if (typeof part === "string" && typeof parts[last] === "string") parts[last] = parts[last] + part
            else if (part !== "") parts.push(part)
        }
        for (const node of list) {
            switch (node.kind) {
                case "text":
                    push(node.text)
                    break
                case "count":
                    push(count === undefined ? "#" : speech.number(count))
                    break
                case "slot": {
                    const value = valueOf(node.name)
                    push(typeof value === "number" ? speech.number(value) : value as string | T)
                    break
                }
                case "plural": {
                    const n = countOf(node.name)
                    const branch = node.branches.get(`=${n}`)
                        ?? node.branches.get(speech.category(n, node.ordinal))
                        ?? node.branches.get("other")!
                    walk(branch, n, parts)
                    break
                }
                case "select": {
                    const value = String(valueOf(node.name))
                    walk(node.branches.get(value) ?? node.branches.get("other")!, count, parts)
                    break
                }
                case "tag": {
                    const inner = walk(node.children, count, [])
                    const wrap = values[node.name]
                    if (typeof wrap === "function") push((wrap as Wrap<T>)(inner))
                    else inner.forEach(push)
                    break
                }
            }
        }
        return parts
    }
    return walk(nodes, undefined, [])
}
