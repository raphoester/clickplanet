import {PointerEvent, useEffect, useId, useRef, useState} from 'react'
import {BonusNotice, BonusNoticeEvent, BonusRules, ChargeKind, Charges, slotOf, Switches} from '../../domain/bonus.ts'
import {BonusGuide, FRESH_GUIDE, isLearning, isNew} from '../../domain/bonusGuide.ts'
import BonusIcon from './BonusIcon.tsx'
import Bubble, {BUBBLE_MS} from './Bubble.tsx'
import './Inventory.css'

export const NOTICE_MS = 3500
export const NOT_ENCLOSED_MS = 5000

const STATE_CLEARANCE_PX = 20

const EMPTY_HINT = "Catch a ? box or win a quiz to get one"

export type InventoryProps = {
    charges: Charges
    rules?: BonusRules
    switches: Switches
    onToggle?: (name: keyof Switches) => void
    bombArmed: boolean
    onToggleBomb?: () => void
    onUseRefill?: () => boolean
    onToggleShield?: () => void
    notice?: BonusNoticeEvent
    guide?: BonusGuide
    onUsed?: (kind: ChargeKind) => void
}

type Said = {
    kind: ChargeKind
    text: string
    chip?: string
    ms?: number
}

export default function Inventory({
    charges,
    rules,
    switches,
    onToggle,
    bombArmed,
    onToggleBomb,
    onUseRefill,
    onToggleShield,
    notice,
    guide = FRESH_GUIDE,
    onUsed,
}: InventoryProps) {
    const [said, setSaid] = useState<Said | undefined>()
    const [shownNotice, setShownNotice] = useState(notice?.seq)

    const say = (kind: ChargeKind, text: string, ms = BUBBLE_MS, chip?: string) => setSaid({kind, text, ms, chip})

    if (notice && notice.seq !== shownNotice) {
        setShownNotice(notice.seq)
        setSaid({kind: slotOf(notice.notice), ...noticeSaid(notice.notice, rules)})
    }

    const [chargesSeen, setChargesSeen] = useState(charges)
    if (charges !== chargesSeen) {
        setChargesSeen(charges)
        if (said && amountOf(charges, said.kind) < amountOf(chargesSeen, said.kind)) setSaid(undefined)
    }

    useEffect(() => {
        if (said?.ms === undefined) return
        const timer = setTimeout(() => setSaid(undefined), said.ms)
        return () => clearTimeout(timer)
    }, [said])

    const unsay = (kind: ChargeKind) => setSaid((now) => now?.kind === kind && now.ms === undefined ? undefined : now)

    const pressed = (kind: ChargeKind, turnsOn: boolean, learnText?: string) => {
        if (turnsOn && learnText && isLearning(guide, kind)) say(kind, learnText)
        else setSaid(undefined)
        if (turnsOn) onUsed?.(kind)
    }

    const active = bombArmed || switches.spread || switches.enclose || switches.shield
    const max = rules?.enclosureMaxTiles

    const slot = (kind: ChargeKind) => ({
        kind,
        guide,
        said: said?.kind === kind ? said : undefined,
        onHint: (text: string) => setSaid((now) => now?.ms !== undefined ? now : {kind, text}),
        onUnhint: () => unsay(kind),
        onEmpty: () => say(kind, EMPTY_HINT),
    })

    return <section className={active ? "inventory inventory--active" : "inventory"} aria-label="Inventory">
        <div className="inventory-slots">
            <Slot {...slot("refill")}
                  name="Refill"
                  held={charges.refill}
                  hint="Fill your clicks to full"
                  onPress={onUseRefill && (() => {
                      onUsed?.("refill")
                      if (onUseRefill()) setSaid(undefined)
                      else say("refill", "Your clicks are already full", NOTICE_MS, "Full")
                  })}/>
            <Slot {...slot("bomb")}
                  name="Bomb"
                  held={charges.bomb}
                  on={bombArmed}
                  state={bombArmed ? "Aim" : undefined}
                  hint={bombArmed ? "Put the bomb away (Esc)" : "Aim the bomb, then hold on the planet to drop it"}
                  onPress={onToggleBomb && (() => {
                      pressed("bomb", !bombArmed, "Hold your finger still on the planet to drop it")
                      onToggleBomb()
                  })}/>
            <Slot {...slot("spreadClicks")}
                  name="Spread"
                  held={charges.spreadClicksLeft > 0}
                  count={countOf(charges.spreadClicksLeft, rules?.spreadClicks)}
                  on={switches.spread}
                  state={switches.spread ? "On" : undefined}
                  hint={switches.spread ? "Switch spread off" : "Switch spread on: each click also takes the tiles around it"}
                  onPress={onToggle && (() => {
                      pressed("spreadClicks", !switches.spread, "Each click now also takes the tiles around it")
                      onToggle("spread")
                  })}/>
            <Slot {...slot("encloseClicks")}
                  name="Enclose"
                  held={charges.enclosures > 0}
                  count={countOf(charges.enclosures, rules?.enclosures)}
                  on={switches.enclose}
                  state={switches.enclose ? "On" : undefined}
                  hint={switches.enclose
                      ? "Switch enclose off"
                      : "Switch enclose on: close a shape of your tiles to take the tiles inside"}
                  onPress={onToggle && (() => {
                      pressed("encloseClicks", !switches.enclose, max
                          ? `Tap your tiles in a ring. The inside becomes yours, up to ${max} tiles`
                          : "Tap your tiles in a ring. The inside becomes yours")
                      onToggle("enclose")
                  })}/>
            <Slot {...slot("shields")}
                  name="Shield"
                  held={charges.shields > 0}
                  count={countOf(charges.shields, rules?.shields)}
                  on={switches.shield}
                  state={switches.shield ? "On" : undefined}
                  hint={switches.shield ? "Switch shield off" : "Switch shield on, then tap your tiles to shield them"}
                  onPress={onToggleShield && (() => {
                      pressed("shields", !switches.shield, "Tap one of your tiles to shield it")
                      onToggleShield()
                  })}/>
        </div>
    </section>
}

function noticeSaid(notice: BonusNotice, rules: BonusRules | undefined): Omit<Said, "kind"> {
    switch (notice) {
        case "shieldFull":
            return {text: "That tile holds all the shields it can", chip: "Full", ms: NOTICE_MS}
        case "shieldTaken":
            return {text: "Tile taken. Tap it again to shield it", ms: NOTICE_MS}
        case "shieldNotYours":
            return {text: "Shields go on your own tiles", ms: NOTICE_MS}
        case "nothingEnclosed": {
            const max = rules?.enclosureMaxTiles
            return {
                text: max ? `Shape is not closed or is too big (${max} tiles max)` : "Shape is not closed or is too big",
                ms: NOT_ENCLOSED_MS,
            }
        }
    }
}

type SlotProps = {
    kind: ChargeKind
    name: string
    held: boolean
    count?: string
    on?: boolean
    state?: string
    hint: string
    guide: BonusGuide
    said?: Said
    onPress?: () => void
    onHint: (text: string) => void
    onUnhint: () => void
    onEmpty: () => void
}

function Slot({kind, name, held, count, on, state, hint, guide, said, onPress, onHint, onUnhint, onEmpty}: SlotProps) {
    const button = useRef<HTMLButtonElement>(null)
    const described = useId()
    const usable = held && onPress !== undefined
    const fresh = usable && isNew(guide, kind)
    const shownState = said?.chip ?? state ?? (fresh ? "New" : undefined)

    const className = [
        "inventory-slot",
        "panel-box",
        `inventory-slot--${kind}`,
        !held && "inventory-slot--empty",
        on && "inventory-slot--on",
        fresh && !state && "inventory-slot--new",
    ].filter(Boolean).join(" ")

    const label = [name, count, shownState].filter(Boolean).join(", ")
    const description = held ? hint : EMPTY_HINT

    const mouse = (event: PointerEvent) => event.pointerType === "mouse"

    return <button type="button"
                   ref={button}
                   className={className}
                   aria-label={label}
                   aria-pressed={on}
                   aria-describedby={described}
                   aria-disabled={!usable || undefined}
                   disabled={held && !onPress}
                   onPointerEnter={(event) => mouse(event) && onHint(description)}
                   onPointerLeave={(event) => mouse(event) && onUnhint()}
                   onClick={usable ? onPress : onEmpty}>
        <span className="inventory-box"><BonusIcon kind={kind}/></span>
        {shownState && <span className="chip inventory-state" aria-hidden="true">{shownState}</span>}
        {count && <span className="inventory-count" aria-hidden="true">{count}</span>}
        <span className="inventory-name" aria-hidden="true">{name}</span>
        <span id={described} hidden>{description}</span>
        {said && <Bubble anchor={button} gap={STATE_CLEARANCE_PX}>{said.text}</Bubble>}
    </button>
}

function amountOf(charges: Charges, kind: ChargeKind): number {
    switch (kind) {
        case "refill":
            return charges.refill ? 1 : 0
        case "bomb":
            return charges.bomb ? 1 : 0
        case "spreadClicks":
            return charges.spreadClicksLeft
        case "encloseClicks":
            return charges.enclosures
        case "shields":
            return charges.shields
    }
}

function countOf(held: number, most: number | undefined): string {
    return most ? `${held}/${most}` : String(held)
}
