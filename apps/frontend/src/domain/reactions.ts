import type {ChatMessage, Reaction, ReactionCount, ReactionsChange} from "../backends/chat.ts";

/**
 * A message's reactions as the stream sends them. The stream is nobody's, so it
 * says nothing about `mine`: what the log already knew is kept.
 */
export function mergedReactions(previous: readonly ReactionCount[], incoming: readonly ReactionCount[]): ReactionCount[] {
    return incoming.map(count => ({
        ...count,
        mine: previous.find(each => each.reaction === count.reaction)?.mine ?? false,
    }))
}

/** `counts` once this player put `reaction` on, or took it off, before the server says so. */
export function toggledReactions(
    counts: readonly ReactionCount[],
    reaction: Reaction,
    on: boolean,
): ReactionCount[] {
    const index = counts.findIndex(each => each.reaction === reaction)
    if (index === -1) return on ? [...counts, {reaction, count: 1, mine: true}] : [...counts]

    const current = counts[index]
    if (current.mine === on) return [...counts]

    const count = current.count + (on ? 1 : -1)
    if (count <= 0) return counts.filter((_, each) => each !== index)

    return counts.map((each, at) => at === index ? {...each, count, mine: on} : each)
}

/** The log with one message's reactions replaced. A message the log does not hold is ignored. */
export function withReactions(
    log: readonly ChatMessage[],
    messageId: string,
    reactions: (current: readonly ReactionCount[]) => ReactionCount[],
): ChatMessage[] {
    if (!log.some(message => message.id === messageId)) return log as ChatMessage[]

    return log.map(message => message.id === messageId
        ? {...message, reactions: reactions(message.reactions)}
        : message)
}

/** The log once a change from the stream landed. */
export function applyReactionsChange(log: readonly ChatMessage[], change: ReactionsChange): ChatMessage[] {
    return withReactions(log, change.messageId, current => mergedReactions(current, change.reactions))
}
