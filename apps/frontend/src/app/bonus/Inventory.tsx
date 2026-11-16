import {PointerEvent, ReactNode, RefObject, useContext, useEffect, useId, useMemo, useRef, useState} from 'react'
import {createPortal} from 'react-dom'
import {
    BonusNotice,
    BonusRules,
    CHARGE_KINDS,
    ChargeKind,
    Charges,
    gatesOf,
    inUnlockOrder,
    lockedAt,
    NEVER,
    NO_LEVEL,
    slotOf,
    Switches,
} from '../../domain/bonus.ts'
import {BonusGuide, isLearning, isNew} from '../../domain/bonusGuide.ts'
import {t, translator} from '../../i18n/page.ts'
import {useBonusGuide} from './bonusGuideStore.ts'
import {useBonus} from './bonusStore.ts'
import {useLevel} from '../level/levelStore.ts'
import {useScreen} from '../hud/screenStore.ts'
import {useCompact} from '../compact.ts'
import BonusIcon from '../components/BonusIcon.tsx'
import Bubble, {BUBBLE_MS} from '../components/Bubble.tsx'
import {LockIcon} from '../components/icons.tsx'
import {useEscape} from '../components/useDialog.ts'
import {DockTray} from '../components/dockTray.ts'
import './Inventory.css'

export const NOTICE_MS = 3500
export const NOT_ENCLOSED_MS = 5000

// The held kinds the bar shows beside All; the others wait in the tray.
export const PHONE_BAR_ROOM = 3
export const DESKTOP_BAR_ROOM = 4

const STATE_CLEARANCE_PX = 20

export type InventoryProps = {
    onToggle?: (name: keyof Switches) => void
    onToggleBomb?: () => void
    onToggleVirus?: () => void
    onUseRefill?: () => boolean
    onToggleShield?: () => void
}

type Place = "bar" | "tray"

type Said = {
    kind: ChargeKind
    text: string
    chip?: string
    ms?: number
    place?: Place
}

type Shown = {
    kind: ChargeKind
    name: string
    held: boolean
    count?: string
    endless?: boolean
    on?: boolean
    state?: string
    hint: string
    onPress?: () => void
    lockedAt?: number
}

export default function Inventory({onToggle, onToggleBomb, onToggleVirus, onUseRefill, onToggleShield}: InventoryProps) {
    const notice = useBonus((bonus) => bonus.notice)
    const charges = useBonus((bonus) => bonus.charges)
    const rules = useBonus((bonus) => bonus.rules)
    const switches = useBonus((bonus) => bonus.switches)
    const bombArmed = useBonus((bonus) => bonus.bombArmed)
    const virusAimed = useBonus((bonus) => bonus.virusAimed)
    const level = useLevel((store) => store.own?.level ?? NO_LEVEL)
    const unlocks = useLevel((store) => store.unlocks)
    const ladder = useMemo(() => ({level, unlocks}), [level, unlocks])
    const open = useScreen((screen) => screen.popovers.bonuses)
    const togglePopover = useScreen((screen) => screen.togglePopover)
    const closePopover = useScreen((screen) => screen.closePopover)
    const compact = useCompact()
    const {guide, markUsed} = useBonusGuide()
    const [said, setSaid] = useState<Said | undefined>()
    const [shownNotice, setShownNotice] = useState(notice?.seq)
    const bag = useRef<HTMLButtonElement>(null)
    const trayId = useId()

    const say = (kind: ChargeKind, text: string, ms = BUBBLE_MS, chip?: string, place?: Place) =>
        setSaid({kind, text, ms, chip, place})

    if (notice && notice.seq !== shownNotice) {
        setShownNotice(notice.seq)
        setSaid({kind: slotOf(notice.notice), ...noticeSaid(notice.notice, rules)})
    }

    const [chargesSeen, setChargesSeen] = useState(charges)
    if (charges !== chargesSeen) {
        setChargesSeen(charges)
        if (said && amountOf(charges, said.kind) < amountOf(chargesSeen, said.kind)) setSaid(undefined)
    }

    if (!open && said?.place === "tray") setSaid(undefined)

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

    const closeTray = () => closePopover("bonuses")

    const active = bombArmed || virusAimed || switches.spread || switches.enclose || switches.shield
    const max = rules?.enclosureMaxTiles
    const unlimited = rules?.enclosuresUnlimited ?? false
    const spreadUnlimited = rules?.spreadUnlimited ?? false
    const gates = gatesOf(rules, ladder)

    const shownOf = (kind: ChargeKind): Omit<Shown, "lockedAt"> | undefined => {
        switch (kind) {
            case "refill":
                return {
                    kind,
                    name: t("bonus.refill"),
                    held: charges.refill,
                    hint: t("inventory.refill.hint"),
                    onPress: onUseRefill && (() => {
                        markUsed("refill")
                        if (onUseRefill()) setSaid(undefined)
                        else say("refill", t("inventory.refill.alreadyFull"), NOTICE_MS, t("inventory.state.full"))
                    }),
                }
            case "bomb":
                if (rules?.bombsOff) return undefined
                return {
                    kind,
                    name: t("bonus.bomb"),
                    held: charges.bomb,
                    on: bombArmed,
                    state: bombArmed ? t("inventory.state.aim") : undefined,
                    hint: bombArmed ? t("inventory.bomb.putAway") : t("inventory.bomb.hint"),
                    onPress: onToggleBomb && (() => {
                        pressed("bomb", !bombArmed, t("inventory.bomb.learn"))
                        onToggleBomb()
                    }),
                }
            case "spreadClicks":
                return {
                    kind,
                    name: t("bonus.spread"),
                    held: spreadUnlimited || charges.spreadClicksLeft > 0,
                    count: spreadUnlimited ? "∞" : countOf(charges.spreadClicksLeft, rules?.spreadClicks),
                    endless: spreadUnlimited,
                    on: switches.spread,
                    state: switches.spread ? t("inventory.state.on") : undefined,
                    hint: switches.spread ? t("inventory.spread.off") : t("bonus.spread.switchOn"),
                    onPress: onToggle && (() => {
                        pressed("spreadClicks", !switches.spread, t("inventory.spread.learn"))
                        onToggle("spread")
                    }),
                }
            case "encloseClicks":
                if (rules?.enclosuresOff) return undefined
                return {
                    kind,
                    name: t("bonus.enclose"),
                    held: unlimited || charges.enclosures > 0,
                    count: unlimited ? "∞" : countOf(charges.enclosures, rules?.enclosures),
                    endless: unlimited,
                    on: switches.enclose,
                    state: switches.enclose ? t("inventory.state.on") : undefined,
                    hint: switches.enclose ? t("inventory.enclose.off") : t("inventory.enclose.hint"),
                    onPress: onToggle && (() => {
                        pressed("encloseClicks", !switches.enclose, max
                            ? t("inventory.enclose.learnUpTo", {max})
                            : t("inventory.enclose.learn"))
                        onToggle("enclose")
                    }),
                }
            case "shields":
                return {
                    kind,
                    name: t("bonus.shield"),
                    held: charges.shields > 0,
                    count: countOf(charges.shields, rules?.shields),
                    on: switches.shield,
                    state: switches.shield ? t("inventory.state.on") : undefined,
                    hint: switches.shield ? t("inventory.shield.off") : t("inventory.shield.hint"),
                    onPress: onToggleShield && (() => {
                        pressed("shields", !switches.shield, t("inventory.shield.learn"))
                        onToggleShield()
                    }),
                }
            case "virus": {
                const most = rules?.virusInfections
                return {
                    kind,
                    name: t("bonus.virus"),
                    held: charges.virus,
                    on: virusAimed,
                    state: virusAimed ? t("inventory.state.aim") : undefined,
                    hint: virusAimed
                        ? t("inventory.virus.putAway")
                        : most ? t("inventory.virus.hintUpTo", {most}) : t("inventory.virus.hint"),
                    onPress: onToggleVirus && (() => {
                        pressed("virus", !virusAimed, most
                            ? t("inventory.virus.learnUpTo", {most})
                            : t("inventory.virus.learn"))
                        onToggleVirus()
                    }),
                }
            }
        }
    }

    const every: Shown[] = inUnlockOrder(CHARGE_KINDS, gates).flatMap((kind) => {
        const shown = shownOf(kind)
        if (!shown) return []
        return [{...shown, lockedAt: shown.held ? undefined : lockedAt(kind, gates, ladder.level)}]
    })
    const held = every.filter((shown) => shown.held)
    const room = compact ? PHONE_BAR_ROOM : DESKTOP_BAR_ROOM
    const onBar = held.slice(0, room)
    const tucked = held.slice(room)

    const saidAt: Place | undefined = said && (open && said.place === "tray" ? "tray" : "bar")
    const shownSaid = (kind: ChargeKind, place: Place) => said?.kind === kind && saidAt === place ? said : undefined

    const slot = (shown: Shown, place: Place) => <Slot key={shown.kind}
                                                       {...shown}
                                                       guide={guide}
                                                       said={shownSaid(shown.kind, place)}
                                                       onPress={place === "tray" && shown.onPress
                                                           ? () => {
                                                               shown.onPress?.()
                                                               bag.current?.focus()
                                                               closeTray()
                                                           }
                                                           : shown.onPress}
                                                       onHint={(text) => setSaid((now) => now?.ms !== undefined ? now : {kind: shown.kind, text, place})}
                                                       onUnhint={() => unsay(shown.kind)}
                                                       onEmpty={() => say(shown.kind, t("inventory.empty"), BUBBLE_MS, undefined, place)}
                                                       onLocked={(at) => say(shown.kind, lockedLine(at), BUBBLE_MS, undefined, place)}/>

    const bagSaid = said && saidAt === "bar" && !onBar.some((shown) => shown.kind === said.kind) ? said : undefined

    return <section className={active ? "inventory inventory--active" : "inventory"} aria-label={t("inventory.label")}>
        <div className="inventory-slots">
            {onBar.map((shown) => slot(shown, "bar"))}
            <Bag button={bag}
                 open={open}
                 trayId={trayId}
                 tucked={tucked}
                 guide={guide}
                 said={bagSaid}
                 onPress={() => togglePopover("bonuses")}/>
        </div>
        {open && <Tray id={trayId} bag={bag}>
            {every.map((shown) => slot(shown, "tray"))}
        </Tray>}
    </section>
}

function Tray({id, bag, children}: {id: string, bag: RefObject<HTMLButtonElement>, children: ReactNode}) {
    const panel = useRef<HTMLDivElement>(null)
    const closePopover = useScreen((screen) => screen.closePopover)
    const host = useContext(DockTray)

    useEscape(() => {
        if (panel.current?.contains(document.activeElement)) bag.current?.focus()
        closePopover("bonuses")
    })

    useEffect(() => {
        const outside = (event: globalThis.PointerEvent) => {
            const target = event.target as Node | null
            if (panel.current?.contains(target) || bag.current?.contains(target)) return
            closePopover("bonuses")
        }
        document.addEventListener("pointerdown", outside)
        return () => document.removeEventListener("pointerdown", outside)
    }, [bag, closePopover])

    const tray = <div id={id} ref={panel} className="inventory-tray" role="group" aria-label={t("inventory.all.label")}>
        <div className="inventory-tray-slots">{children}</div>
    </div>
    return host ? createPortal(tray, host) : tray
}

type BagProps = {
    button: RefObject<HTMLButtonElement>
    open: boolean
    trayId: string
    tucked: readonly Shown[]
    guide: BonusGuide
    said?: Said
    onPress: () => void
}

function Bag({button, open, trayId, tucked, guide, said, onPress}: BagProps) {
    const busy = tucked.find((shown) => shown.on)
    const fresh = tucked.some((shown) => shown.onPress && !shown.endless && isNew(guide, shown.kind))
    const state = busy?.state ?? (fresh ? t("inventory.state.new") : undefined)

    const className = [
        "inventory-slot",
        "panel-box",
        "inventory-bag",
        busy && "inventory-slot--on",
        fresh && !busy && "inventory-slot--new",
    ].filter(Boolean).join(" ")

    const label = tucked.length > 0
        ? t("inventory.all.more", {count: tucked.length})
        : t("inventory.all.label")

    return <button type="button"
                   ref={button}
                   className={className}
                   aria-label={state ? t("inventory.slot.state", {name: label, state}) : label}
                   aria-expanded={open}
                   aria-controls={open ? trayId : undefined}
                   onClick={onPress}>
        <span className="inventory-box"><AllIcon/></span>
        {state && <span className="chip inventory-state" aria-hidden="true">{state}</span>}
        {tucked.length > 0 && <span className="inventory-count" aria-hidden="true">
            {t("inventory.more", {count: translator().number(tucked.length)})}
        </span>}
        <span className="inventory-name" aria-hidden="true">{t("inventory.all")}</span>
        {said && <Bubble anchor={button} gap={STATE_CLEARANCE_PX}>{said.text}</Bubble>}
    </button>
}

function AllIcon() {
    return <svg className="bonus-icon" viewBox="0 0 48 48" aria-hidden="true">
        <rect className="inventory-all inventory-all--refill" x="5" y="5" width="17" height="17" rx="4"/>
        <rect className="inventory-all inventory-all--spread" x="26" y="5" width="17" height="17" rx="4"/>
        <rect className="inventory-all inventory-all--enclose" x="5" y="26" width="17" height="17" rx="4"/>
        <rect className="inventory-all inventory-all--shield" x="26" y="26" width="17" height="17" rx="4"/>
    </svg>
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
        case "virusNoEnemy":
            return {text: t("inventory.notice.virusNoEnemy"), ms: NOTICE_MS}
        case "nothingEnclosed": {
            const max = rules?.enclosureMaxTiles
            return {
                text: max ? t("inventory.notice.notEnclosedMax", {max}) : t("inventory.notice.notEnclosed"),
                ms: NOT_ENCLOSED_MS,
            }
        }
    }
}

type SlotProps = Shown & {
    guide: BonusGuide
    said?: Said
    onHint: (text: string) => void
    onUnhint: () => void
    onEmpty: () => void
    onLocked: (level: number) => void
}

function Slot({kind, name, held, count, endless, on, state, hint, lockedAt, guide, said, onPress, onHint, onUnhint, onEmpty, onLocked}: SlotProps) {
    const button = useRef<HTMLButtonElement>(null)
    const described = useId()
    const usable = held && onPress !== undefined
    const fresh = usable && !endless && isNew(guide, kind)
    const shownState = said?.chip ?? state ?? (fresh ? t("inventory.state.new") : undefined)

    const className = [
        "inventory-slot",
        "panel-box",
        `inventory-slot--${kind}`,
        !held && !lockedAt && "inventory-slot--empty",
        lockedAt && "inventory-slot--locked",
        on && "inventory-slot--on",
        fresh && !state && "inventory-slot--new",
    ].filter(Boolean).join(" ")

    const label = lockedAt ? lockedLabel(name, lockedAt) : labelOf(name, count, shownState)
    const description = held ? hint : lockedAt ? lockedLine(lockedAt) : t("inventory.empty")

    const mouse = (event: PointerEvent) => event.pointerType === "mouse"

    const press = () => {
        if (held && onPress) onPress()
        else if (lockedAt) onLocked(lockedAt)
        else onEmpty()
    }

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
                   onClick={press}>
        {endless && on && <svg className="inventory-loop" aria-hidden="true"><rect width="100%" height="100%"/></svg>}
        <span className="inventory-box"><BonusIcon kind={kind}/></span>
        {shownState && <span className="chip inventory-state" aria-hidden="true">{shownState}</span>}
        {lockedAt
            ? <span className="inventory-count inventory-lock" aria-hidden="true">
                <LockIcon size={11}/>{lockedAt !== NEVER && translator().number(lockedAt)}
            </span>
            : count && <span className="inventory-count" aria-hidden="true">{count}</span>}
        <span className="inventory-name" aria-hidden="true">{name}</span>
        <span id={described} hidden>{description}</span>
        {said && <Bubble anchor={button} gap={STATE_CLEARANCE_PX}>{said.text}</Bubble>}
    </button>
}

function lockedLabel(name: string, level: number): string {
    return level === NEVER ? t("inventory.slot.lockedForNow", {name}) : t("inventory.slot.locked", {name, level})
}

function lockedLine(level: number): string {
    return level === NEVER ? t("inventory.lockedForNow") : t("inventory.locked", {level})
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
        case "virus":
            return charges.virus ? 1 : 0
    }
}

function countOf(held: number, most: number | undefined): string {
    return most ? t("inventory.heldOfMost", {held, most}) : translator().number(held)
}
