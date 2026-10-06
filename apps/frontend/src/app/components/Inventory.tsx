import {useEffect, useState} from 'react'
import {BonusRules, ChargeKind, Charges, Switches} from '../../domain/bonus.ts'
import BonusIcon from './BonusIcon.tsx'
import './Inventory.css'

const FULL_NOTICE_MS = 2000

export type InventoryProps = {
    charges: Charges
    rules?: BonusRules
    switches: Switches
    onToggle?: (name: keyof Switches) => void
    bombArmed: boolean
    onToggleBomb?: () => void
    onUseRefill?: () => boolean
    onToggleShield?: () => void
    shieldFull?: number
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
    shieldFull = 0,
}: InventoryProps) {
    const [full, setFull] = useState(false)
    const [fullSeen, setFullSeen] = useState(shieldFull)
    const tileFull = shieldFull !== fullSeen

    useEffect(() => {
        if (!full) return
        const timer = setTimeout(() => setFull(false), FULL_NOTICE_MS)
        return () => clearTimeout(timer)
    }, [full])

    useEffect(() => {
        if (!tileFull) return
        const timer = setTimeout(() => setFullSeen(shieldFull), FULL_NOTICE_MS)
        return () => clearTimeout(timer)
    }, [tileFull, shieldFull])

    const active = bombArmed || switches.spread || switches.enclose || switches.shield

    return <section className={active ? "inventory inventory--active" : "inventory"} aria-label="Inventory">
        <div className="inventory-slots">
            <Slot kind="refill"
                  name="Refill"
                  held={charges.refill}
                  state={full ? "Full" : undefined}
                  hint={full ? "Your clicks are already full" : "Fill your clicks to full"}
                  onPress={onUseRefill && (() => setFull(!onUseRefill()))}/>
            <Slot kind="bomb"
                  name="Bomb"
                  held={charges.bomb}
                  on={bombArmed}
                  state={bombArmed ? "Aim" : undefined}
                  hint={bombArmed ? "Put the bomb away (Esc)" : "Aim the bomb, then hold on the planet to drop it"}
                  onPress={onToggleBomb}/>
            <Slot kind="spreadClicks"
                  name="Spread"
                  held={charges.spreadClicksLeft > 0}
                  count={countOf(charges.spreadClicksLeft, rules?.spreadClicks)}
                  on={switches.spread}
                  state={switches.spread ? "On" : undefined}
                  hint={switches.spread ? "Switch spread off" : "Switch spread on: each click also takes the tiles around it"}
                  onPress={onToggle && (() => onToggle("spread"))}/>
            <Slot kind="encloseClicks"
                  name="Enclose"
                  held={charges.enclosures > 0}
                  count={countOf(charges.enclosures, rules?.enclosures)}
                  on={switches.enclose}
                  state={switches.enclose ? "On" : undefined}
                  hint={switches.enclose
                      ? "Switch enclose off"
                      : "Switch enclose on: close a shape of your tiles to take the tiles inside"}
                  onPress={onToggle && (() => onToggle("enclose"))}/>
            <Slot kind="shields"
                  name="Shield"
                  held={charges.shields > 0}
                  count={countOf(charges.shields, rules?.shields)}
                  on={switches.shield}
                  state={tileFull ? "Full" : switches.shield ? "On" : undefined}
                  hint={tileFull ? "That tile is full" : switches.shield ? "Switch shield off" : "Switch shield on"}
                  onPress={onToggleShield}/>
        </div>
    </section>
}

function Slot({kind, name, held, count, on, state, hint, onPress}: {
    kind: ChargeKind
    name: string
    held: boolean
    count?: string
    on?: boolean
    state?: string
    hint: string
    onPress?: () => void
}) {
    const className = [
        "inventory-slot",
        "panel-box",
        `inventory-slot--${kind}`,
        !held && "inventory-slot--empty",
        on && "inventory-slot--on",
    ].filter(Boolean).join(" ")

    const label = [name, count, state].filter(Boolean).join(", ")

    return <button type="button"
                   className={className}
                   aria-label={label}
                   aria-pressed={on}
                   title={held ? hint : `No ${name.toLowerCase()} held. Catch a box`}
                   disabled={!held || !onPress}
                   onClick={onPress}>
        <span className="inventory-box"><BonusIcon kind={kind}/></span>
        {state && <span className="chip inventory-state" aria-hidden="true">{state}</span>}
        {count && <span className="inventory-count" aria-hidden="true">{count}</span>}
        <span className="inventory-name" aria-hidden="true">{name}</span>
    </button>
}

function countOf(held: number, most: number | undefined): string {
    return most ? `${held}/${most}` : String(held)
}
