import {PointerEvent, useEffect, useId, useRef, useState} from 'react'
import {BonusNotice, BonusRules, ChargeKind, Charges, slotOf, Switches} from '../../domain/bonus.ts'
import {BonusGuide, isLearning, isNew} from '../../domain/bonusGuide.ts'
import {t, translator} from '../../i18n/page.ts'
import {useBonusGuide} from './bonusGuideStore.ts'
import {useBonus} from './bonusStore.ts'
import BonusIcon from '../components/BonusIcon.tsx'
import Bubble, {BUBBLE_MS} from '../components/Bubble.tsx'
import './Inventory.css'

export const NOTICE_MS = 3500
export const NOT_ENCLOSED_MS = 5000

const STATE_CLEARANCE_PX = 20

export type InventoryProps = {
    onToggle?: (name: keyof Switches) => void
    onToggleBomb?: () => void
    onUseRefill?: () => boolean
    onToggleShield?: () => void
}

type Said = {
    kind: ChargeKind
    text: string
    chip?: string
    ms?: number
}

export default function Inventory({onToggle, onToggleBomb, onUseRefill, onToggleShield}: InventoryProps) {
    const notice = useBonus((bonus) => bonus.notice)
    const charges = useBonus((bonus) => bonus.charges)
    const rules = useBonus((bonus) => bonus.rules)
    const switches = useBonus((bonus) => bonus.switches)
    const bombArmed = useBonus((bonus) => bonus.bombArmed)
    const {guide, markUsed} = useBonusGuide()
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
        if (turnsOn) markUsed(kind)
    }

    const active = bombArmed || switches.spread || switches.enclose || switches.shield
    const max = rules?.enclosureMaxTiles
    const unlimited = rules?.enclosuresUnlimited ?? false
    const spreadUnlimited = rules?.spreadUnlimited ?? false

    const slot = (kind: ChargeKind) => ({
        kind,
        guide,
        said: said?.kind === kind ? said : undefined,
        onHint: (text: string) => setSaid((now) => now?.ms !== undefined ? now : {kind, text}),
        onUnhint: () => unsay(kind),
        onEmpty: () => say(kind, t("inventory.empty")),
    })

    return <section className={active ? "inventory inventory--active" : "inventory"} aria-label={t("inventory.label")}>
        <div className="inventory-slots">
            <Slot {...slot("refill")}
                  name={t("bonus.refill")}
                  held={charges.refill}
                  hint={t("inventory.refill.hint")}
                  onPress={onUseRefill && (() => {
                      markUsed("refill")
                      if (onUseRefill()) setSaid(undefined)
                      else say("refill", t("inventory.refill.alreadyFull"), NOTICE_MS, t("inventory.state.full"))
                  })}/>
            {!rules?.bombsOff && <Slot {...slot("bomb")}
                  name={t("bonus.bomb")}
                  held={charges.bomb}
                  on={bombArmed}
                  state={bombArmed ? t("inventory.state.aim") : undefined}
                  hint={bombArmed ? t("inventory.bomb.putAway") : t("inventory.bomb.hint")}
                  onPress={onToggleBomb && (() => {
                      pressed("bomb", !bombArmed, t("inventory.bomb.learn"))
                      onToggleBomb()
                  })}/>}
            <Slot {...slot("spreadClicks")}
                  name={t("bonus.spread")}
                  held={spreadUnlimited || charges.spreadClicksLeft > 0}
                  count={spreadUnlimited ? "∞" : countOf(charges.spreadClicksLeft, rules?.spreadClicks)}
                  endless={spreadUnlimited}
                  on={switches.spread}
                  state={switches.spread ? t("inventory.state.on") : undefined}
                  hint={switches.spread ? t("inventory.spread.off") : t("bonus.spread.switchOn")}
                  onPress={onToggle && (() => {
                      pressed("spreadClicks", !switches.spread, t("inventory.spread.learn"))
                      onToggle("spread")
                  })}/>
            {!rules?.enclosuresOff && <Slot {...slot("encloseClicks")}
                  name={t("bonus.enclose")}
                  held={unlimited || charges.enclosures > 0}
                  count={unlimited ? "∞" : countOf(charges.enclosures, rules?.enclosures)}
                  endless={unlimited}
                  on={switches.enclose}
                  state={switches.enclose ? t("inventory.state.on") : undefined}
                  hint={switches.enclose ? t("inventory.enclose.off") : t("inventory.enclose.hint")}
                  onPress={onToggle && (() => {
                      pressed("encloseClicks", !switches.enclose, max
                          ? t("inventory.enclose.learnUpTo", {max})
                          : t("inventory.enclose.learn"))
                      onToggle("enclose")
                  })}/>}
            <Slot {...slot("shields")}
                  name={t("bonus.shield")}
                  held={charges.shields > 0}
                  count={countOf(charges.shields, rules?.shields)}
                  on={switches.shield}
                  state={switches.shield ? t("inventory.state.on") : undefined}
                  hint={switches.shield ? t("inventory.shield.off") : t("inventory.shield.hint")}
                  onPress={onToggleShield && (() => {
                      pressed("shields", !switches.shield, t("inventory.shield.learn"))
                      onToggleShield()
                  })}/>
        </div>
    </section>
}

function noticeSaid(notice: BonusNotice, rules: BonusRules | undefined): Omit<Said, "kind"> {
    switch (notice) {
        case "shieldFull":
            return {text: t("inventory.notice.shieldFull"), chip: t("inventory.state.full"), ms: NOTICE_MS}
        case "shieldCapital":
            return {text: t("inventory.notice.shieldCapital"), ms: NOTICE_MS}
        case "shieldTaken":
            return {text: t("inventory.notice.shieldTaken"), ms: NOTICE_MS}
        case "shieldNotYours":
            return {text: t("inventory.notice.shieldNotYours"), ms: NOTICE_MS}
        case "nothingEnclosed": {
            const max = rules?.enclosureMaxTiles
            return {
                text: max ? t("inventory.notice.notEnclosedMax", {max}) : t("inventory.notice.notEnclosed"),
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
    endless?: boolean
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

function Slot({kind, name, held, count, endless, on, state, hint, guide, said, onPress, onHint, onUnhint, onEmpty}: SlotProps) {
    const button = useRef<HTMLButtonElement>(null)
    const described = useId()
    const usable = held && onPress !== undefined
    const fresh = usable && !endless && isNew(guide, kind)
    const shownState = said?.chip ?? state ?? (fresh ? t("inventory.state.new") : undefined)

    const className = [
        "inventory-slot",
        "panel-box",
        `inventory-slot--${kind}`,
        !held && "inventory-slot--empty",
        on && "inventory-slot--on",
        fresh && !state && "inventory-slot--new",
    ].filter(Boolean).join(" ")

    const label = labelOf(name, count, shownState)
    const description = held ? hint : t("inventory.empty")

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
        {endless && on && <svg className="inventory-loop" aria-hidden="true"><rect width="100%" height="100%"/></svg>}
        <span className="inventory-box"><BonusIcon kind={kind}/></span>
        {shownState && <span className="chip inventory-state" aria-hidden="true">{shownState}</span>}
        {count && <span className="inventory-count" aria-hidden="true">{count}</span>}
        <span className="inventory-name" aria-hidden="true">{name}</span>
        <span id={described} hidden>{description}</span>
        {said && <Bubble anchor={button} gap={STATE_CLEARANCE_PX}>{said.text}</Bubble>}
    </button>
}

function labelOf(name: string, count: string | undefined, state: string | undefined): string {
    if (count && state) return t("inventory.slot.countAndState", {name, count, state})
    if (count) return t("inventory.slot.count", {name, count})
    if (state) return t("inventory.slot.state", {name, state})
    return name
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
    return most ? t("inventory.heldOfMost", {held, most}) : translator().number(held)
}
