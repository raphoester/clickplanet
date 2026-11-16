import {ChargesHeld} from "../gen/grpc/planet/v1/planet_pb.ts"
import {Grant, MOST} from "./actions.ts"

// What the operator typed in a count: a whole number from 0 to 99, and 0 for anything else.
export function wholeOf(typed: string): number {
    const n = Math.floor(Number(typed))
    return Number.isFinite(n) && n > 0 ? Math.min(n, MOST) : 0
}

export function grantWords(charges: Grant): string {
    return listOf([
        plural(charges.refills, "refill"),
        plural(charges.bombs, "bomb"),
        plural(charges.enclosures, "enclosure"),
        plural(charges.spreadClicks, "spread click"),
        plural(charges.shields, "shield"),
        plural(charges.viruses, "virus", "viruses"),
        plural(charges.mines, "mine"),
        plural(charges.rallies, "rally", "rallies"),
    ])
}

export function heldWords(held: ChargesHeld): string {
    return listOf([
        plural(held.refills, "refill"),
        plural(held.bombs, "bomb"),
        plural(held.enclosures, "enclosure"),
        plural(held.spreadClicksLeft, "spread click"),
        plural(held.shields, "shield"),
        plural(held.viruses, "virus", "viruses"),
        plural(held.mines, "mine"),
        plural(held.rallies, "rally", "rallies"),
    ])
}

function plural(n: number, noun: string, nouns = `${noun}s`): string | false {
    return n > 0 && (n === 1 ? `1 ${noun}` : `${n} ${nouns}`)
}

function listOf(parts: (string | false)[]): string {
    const said = parts.filter((part): part is string => part !== false)
    if (said.length === 0) return "nothing"
    if (said.length === 1) return said[0]
    return `${said.slice(0, -1).join(", ")} and ${said[said.length - 1]}`
}
