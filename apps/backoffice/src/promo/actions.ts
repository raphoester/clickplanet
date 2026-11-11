import {JsonObject, JsonValue} from "@bufbuild/protobuf"
import {Code, ConnectError} from "@connectrpc/connect"
import {PromoAdmin} from "../clients.ts"
import {GetRulesResponse as BonusesRules} from "../gen/grpc/bonuses/v1/admin_pb.ts"
import {GetRulesResponse as PlanetRules} from "../gen/grpc/planet/v1/admin_pb.ts"
import {CardDraft} from "../gen/grpc/promo/v1/admin_pb.ts"
import {DEFAULT_SET} from "../ops/rules.ts"
import {overlay} from "../ops/overlay.ts"
import {wordsOf} from "../refusal.ts"
import {fill, Values} from "./fill.ts"

export type Put = (url: string, file: Blob, contentType: string, onSent: (bytes: number) => void) => Promise<void>

export const ACCEPTED = "image/jpeg,image/png,image/webp"
export const BACKGROUNDS_OFF = "Backgrounds are off on the server."
export const NOT_IN_BUCKET = "The file is not in the bucket. Upload it again."
export const BACKGROUND_GONE = "This background is no longer kept."

// The backend's rules for a set's cards (promo's cards.NewDeck).
export const MAX_CARDS = 5
export const MAX_HEADLINE = 40
export const MAX_LINE = 120

// `resend`: the bucket does not hold the file, so the next try sends it again.
export class Refused extends Error {
    constructor(words: string, readonly resend: boolean) {
        super(words)
    }
}

// The background's id once the bucket holds the file.
export async function sendBackground(promo: PromoAdmin, put: Put, file: File, onSent: (bytes: number) => void): Promise<string> {
    let link
    try {
        link = await promo.createBackgroundUpload({contentType: file.type, sizeBytes: BigInt(file.size)})
    } catch (err) {
        throw new Refused(wordsOf(err, {failedPrecondition: BACKGROUNDS_OFF}), true)
    }
    try {
        await put(link.uploadUrl, file, file.type, onSent)
    } catch (err) {
        throw new Refused(`The upload failed: ${err instanceof Error ? err.message : String(err)}.`, true)
    }
    return link.backgroundId
}

export async function keepBackground(promo: PromoAdmin, backgroundId: string, name: string): Promise<void> {
    try {
        await promo.keepBackground({backgroundId, name: name.trim()})
    } catch (err) {
        const resend = ConnectError.from(err).code === Code.FailedPrecondition
        throw new Refused(wordsOf(err, {failedPrecondition: NOT_IN_BUCKET}), resend)
    }
}

// The server names the sets whose cards show it.
export function removeRefusalOf(err: unknown): string {
    const error = ConnectError.from(err)
    if (error.code === Code.FailedPrecondition) return `Change the cards that show it first: ${error.rawMessage}.`
    return wordsOf(err, {notFound: BACKGROUND_GONE})
}

// A set planet does not keep, or a background no longer in the library: the server says which.
export function saveRefusalOf(err: unknown): string {
    const error = ConnectError.from(err)
    if (error.code === Code.FailedPrecondition) return `The server refused it: ${error.rawMessage}.`
    return wordsOf(err)
}

// The prefix a card names the offers' values by: `{bonuses.boxes.minInterval}` is bonuses.v1.Rules' boxes.minInterval.
export const OFFERS = "bonuses"

// What each set planet keeps plays, as the server fills a card: planet's values, and under `bonuses` the offers' for the
// set of the same name, each default's with the set's own over them. A set the offers do not keep has no `bonuses`.
export function playedBySet(planet: PlanetRules, offers: BonusesRules): Map<string, JsonValue> {
    const theirs = playedOf(offers.sets)
    return new Map([...playedOf(planet.sets)].map(([name, played]) => {
        const offered = theirs.get(name)
        return [name, offered === undefined ? played : {...(played as JsonObject), [OFFERS]: offered}]
    }))
}

function playedOf(sets: {name: string, rules?: {toJson: () => JsonValue}}[]): Map<string, JsonValue> {
    const defaults = sets.find(set => set.name === DEFAULT_SET)?.rules?.toJson() ?? {}
    return new Map(sets.map(set => [set.name, set.name === DEFAULT_SET ? defaults : overlay(defaults, set.rules?.toJson() ?? {})]))
}

// What the server would refuse of a card, before it is sent.
export function draftProblems(draft: CardDraft, values: Values): string[] {
    const problems: string[] = []
    if (draft.backgroundId === "") problems.push("Pick a background.")
    const headline = draft.headline.trim()
    if (headline === "" || [...headline].length > MAX_HEADLINE) problems.push(`A headline of 1 to ${MAX_HEADLINE} characters.`)
    if ([...draft.line.trim()].length > MAX_LINE) problems.push(`A line of ${MAX_LINE} characters at most.`)
    for (const words of [draft.headline, draft.line]) problems.push(...fill(words, values).problems)
    return [...new Set(problems)]
}
