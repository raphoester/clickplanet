import {useEffect, useRef} from "react";
import {Reaction, ReactionCount} from "../../backends/chat.ts";
import {AddReactionIcon} from "../components/icons.tsx";
import {REACTION_IMAGES} from "./reactionsAsset.ts";

// What belongs to one message's reactions, so a click on it does not count as a click outside.
const OWNER = "data-reactions-for"

export type ReactionBarProps = {
    messageId: string
    reactions: ReactionCount[]
    /** Absent, the counts are shown and nothing can be clicked. */
    onReact?: (reaction: Reaction, on: boolean) => void
    picking: boolean
    setPicking: (picking: boolean) => void
}

/**
 * A message's reactions, and the picker when it is open. Each is drawn from our
 * own images: the system's emoji font looks different, or broken, everywhere.
 */
export default function ReactionBar({messageId, reactions, onReact, picking, setPicking}: ReactionBarProps) {
    const shell = useRef<HTMLDivElement>(null)

    // A click anywhere else closes the picker, and so does Escape.
    useEffect(() => {
        if (!picking) return
        const clicked = (event: PointerEvent) => {
            const owner = (event.target as Element | null)?.closest?.(`[${OWNER}]`)
            if (owner?.getAttribute(OWNER) !== messageId) setPicking(false)
        }
        const pressed = (event: KeyboardEvent) => {
            if (event.key === "Escape") setPicking(false)
        }
        document.addEventListener("pointerdown", clicked)
        document.addEventListener("keydown", pressed)
        return () => {
            document.removeEventListener("pointerdown", clicked)
            document.removeEventListener("keydown", pressed)
        }
    }, [picking, setPicking, messageId])

    const counted = reactions.filter(count => REACTION_IMAGES.has(count.reaction))
    const open = picking && onReact !== undefined

    // Opened on the last message, the picker would sit under the fold of the log.
    useEffect(() => {
        if (open) shell.current?.scrollIntoView?.({block: "nearest"})
    }, [open])

    if (counted.length === 0 && !open) return null

    const react = (reaction: Reaction, on: boolean) => {
        setPicking(false)
        onReact?.(reaction, on)
    }

    return <div className="chat-reactions-shell" ref={shell} {...{[OWNER]: messageId}}>
        {counted.length > 0 && <div className="chat-reactions">
            {counted.map(count => {
                const image = REACTION_IMAGES.get(count.reaction)!
                return <button type="button"
                               key={count.reaction}
                               className={count.mine ? "chat-reaction chat-reaction-mine" : "chat-reaction"}
                               aria-pressed={count.mine}
                               aria-label={`${image.label}: ${count.count}`}
                               title={image.label}
                               disabled={!onReact}
                               onClick={() => react(count.reaction, !count.mine)}>
                    <img src={image.url} alt="" width={16} height={16} draggable={false}/>
                    <span className="chat-reaction-count">{count.count}</span>
                </button>
            })}
        </div>}

        {open && <div className="chat-reaction-picker" role="group" aria-label="Reactions">
            {[...REACTION_IMAGES].map(([reaction, image]) => {
                const mine = reactions.some(count => count.reaction === reaction && count.mine)
                return <button type="button"
                               key={reaction}
                               className="chat-reaction-choice"
                               aria-pressed={mine}
                               aria-label={image.label}
                               title={image.label}
                               onClick={() => react(reaction, !mine)}>
                    <img src={image.url} alt="" width={24} height={24} draggable={false}/>
                </button>
            })}
        </div>}
    </div>
}

export type AddReactionButtonProps = {
    messageId: string
    picking: boolean
    setPicking: (picking: boolean) => void
}

/** Beside the balloon, so a message nobody reacted to takes no more room than before. */
export function AddReactionButton({messageId, picking, setPicking}: AddReactionButtonProps) {
    return <button type="button"
                   className={picking ? "chat-reaction-add chat-reaction-add-open" : "chat-reaction-add"}
                   aria-label="Add a reaction"
                   aria-expanded={picking}
                   title="Add a reaction"
                   onClick={() => setPicking(!picking)}
                   {...{[OWNER]: messageId}}>
        <AddReactionIcon/>
    </button>
}
