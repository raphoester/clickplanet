import {useEffect, useId, useRef, useState} from "react";
import {createPortal} from "react-dom";
import {Reaction, ReactionCount} from "../../backends/chat.ts";
import {whoReacted} from "../../domain/reactions.ts";
import {AddReactionIcon} from "../components/icons.tsx";
import {truncate} from "../truncate.ts";
import {REACTION_IMAGES} from "./reactionsAsset.ts";

const OWNER = "data-reactions-for"

export type ReactionBarProps = {
    messageId: string
    reactions: ReactionCount[]
    onReact?: (reaction: Reaction, on: boolean) => void
    picking: boolean
    setPicking: (picking: boolean) => void
}

export default function ReactionBar({messageId, reactions, onReact, picking, setPicking}: ReactionBarProps) {
    const shell = useRef<HTMLDivElement>(null)

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
            {counted.map(count => <ReactionChip key={count.reaction}
                                                count={count}
                                                onReact={onReact && (on => react(count.reaction, on))}/>)}
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

const WHO_MAX_LENGTH = 18

const WHO_MAX_WIDTH_PX = 220
const WHO_GAP_PX = 8
const WHO_EDGE_PX = 8

const WHO_ROOM_PX = 140

type Anchor = {
    left: number
    top: number
    above: boolean
}

type ReactionChipProps = {
    count: ReactionCount
    onReact?: (on: boolean) => void
}

function ReactionChip({count, onReact}: ReactionChipProps) {
    const chip = useRef<HTMLButtonElement>(null)
    const [at, setAt] = useState<Anchor | undefined>(undefined)
    const described = useId()
    const image = REACTION_IMAGES.get(count.reaction)!

    const show = () => setAt(anchorOf(chip.current))

    useEffect(() => {
        if (!at) return

        const hide = () => setAt(undefined)
        window.addEventListener("scroll", hide, true)
        window.addEventListener("resize", hide)
        return () => {
            window.removeEventListener("scroll", hide, true)
            window.removeEventListener("resize", hide)
        }
    }, [at])

    return <>
        <button type="button"
                ref={chip}
                className={count.mine ? "chat-reaction chat-reaction-mine" : "chat-reaction"}
                aria-pressed={count.mine}
                aria-label={`${image.label}: ${count.count}`}
                aria-describedby={at ? described : undefined}
                disabled={!onReact}
                onPointerEnter={show}
                onPointerLeave={() => setAt(undefined)}
                onFocus={show}
                onBlur={() => setAt(undefined)}
                onClick={() => onReact?.(!count.mine)}>
            <img src={image.url} alt="" width={16} height={16} draggable={false}/>
            <span className="chat-reaction-count">{count.count}</span>
        </button>
        {at && <ReactionWho id={described} at={at} count={count} label={image.label}/>}
    </>
}

function anchorOf(chip: HTMLElement | null): Anchor | undefined {
    const box = chip?.getBoundingClientRect()
    if (!box) return undefined

    const half = WHO_MAX_WIDTH_PX / 2
    const middle = box.left + box.width / 2
    const above = box.top > WHO_ROOM_PX

    return {
        left: Math.min(Math.max(middle, half + WHO_EDGE_PX), window.innerWidth - half - WHO_EDGE_PX),
        top: above ? box.top - WHO_GAP_PX : box.bottom + WHO_GAP_PX,
        above,
    }
}

type ReactionWhoProps = {
    id: string
    at: Anchor
    count: ReactionCount
    label: string
}

function ReactionWho({id, at, count, label}: ReactionWhoProps) {
    const {names, more} = whoReacted(count)

    // On the body: the panel's backdrop-filter would make position: fixed relative to it.
    return createPortal(
        <div id={id}
             role="tooltip"
             className={at.above ? "chat-reaction-who chat-reaction-who-above" : "chat-reaction-who"}
             style={{left: `${at.left}px`, top: `${at.top}px`, maxWidth: `${WHO_MAX_WIDTH_PX}px`}}>
            <span className="chat-reaction-who-label">{label}</span>
            <ul className="chat-reaction-who-names">
                {names.map((name, index) => <li key={`${index}-${name}`}>{truncate(name, WHO_MAX_LENGTH)}</li>)}
                {more > 0 && <li className="chat-reaction-who-more">{restOf(names.length, more)}</li>}
            </ul>
        </div>,
        document.body)
}

function restOf(named: number, more: number): string {
    if (named > 0) return `and ${more} more`
    return more === 1 ? "1 player" : `${more} players`
}

export type AddReactionButtonProps = {
    messageId: string
    picking: boolean
    setPicking: (picking: boolean) => void
}

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
