import {ReactNode, useEffect, useRef} from 'react'
import {ClickBudget, nextClickProgress, now, secondsToOneMore, tokensAt} from "../../backends/clickBudget.ts"
import {factor} from "../../domain/clickPrice.ts"
import {HourglassIcon, UsersIcon} from "./icons.tsx"
import {useEscape} from "./useDialog.ts"
import {offerText, SHARED_WITH} from "./clickOffer.ts"
import "./ClickBudgetMeter.css"

export type ClickBudgetMeterProps = {
    budget?: ClickBudget
    children?: ReactNode
    refusals?: number
    onSignIn?: () => void
    compact?: boolean
    open?: boolean
    onToggleOpen?: () => void
    popover?: ReactNode
}

const MAX_PIPS = 12

const LOW_WATER = 3

const COUNTDOWN_FROM_S = 1.5

const STEP_MS = 250

export default function ClickBudgetMeter({
    budget,
    children,
    refusals = 0,
    onSignIn,
    compact = false,
    open = false,
    onToggleOpen,
    popover,
}: ClickBudgetMeterProps) {
    const root = useRef<HTMLDivElement>(null)
    const count = useRef<HTMLSpanElement>(null)
    const wait = useRef<HTMLSpanElement>(null)

    useEffect(() => {
        const box = root.current
        if (!box || refusals === 0) return

        box.classList.remove("click-budget-refused")
        // Forces a reflow, so a refusal mid-animation restarts it.
        void box.offsetWidth
        box.classList.add("click-budget-refused")

        const done = () => box.classList.remove("click-budget-refused")
        box.addEventListener("animationend", done, {once: true})
        return () => box.removeEventListener("animationend", done)
    }, [refusals])

    useEffect(() => {
        if (!budget) return

        const box = root.current
        const label = count.current
        if (!box || !label) return

        let shown = -1
        let shownWait = ""

        const bar = budget.capacity > MAX_PIPS

        const draw = () => {
            const at = now()
            const tokens = tokensAt(budget, at)
            const whole = Math.floor(tokens)

            box.style.setProperty("--click-budget-tokens", tokens.toFixed(3))

            if (bar) box.style.setProperty("--click-budget-next", nextClickProgress(budget, at).toFixed(3))

            if (wait.current) {
                const text = waitText(budget, at)
                if (text !== shownWait) {
                    shownWait = text
                    wait.current.textContent = text
                }
            }

            if (whole === shown) return
            shown = whole

            label.textContent = String(whole)
            box.setAttribute("aria-valuenow", String(whole))
            box.classList.toggle("click-budget-empty", whole === 0)
            box.classList.toggle("click-budget-low", whole > 0 && whole <= LOW_WATER)
            box.classList.toggle("click-budget-full", whole >= budget.capacity)
        }

        draw()

        if (window.matchMedia?.("(prefers-reduced-motion: reduce)").matches) {
            const timer = setInterval(draw, STEP_MS)
            return () => clearInterval(timer)
        }

        let frame = requestAnimationFrame(function tick() {
            draw()
            frame = requestAnimationFrame(tick)
        })

        return () => cancelAnimationFrame(frame)
    }, [budget])

    if (!budget) return children ? <div className="click-budget-dock panel">{children}</div> : null

    const pips = budget.capacity <= MAX_PIPS ? budget.capacity : 0

    const whole = Math.floor(tokensAt(budget, now()))

    const slowdown = budget.price && budget.price.slowdown > 1 ? budget.price.slowdown : undefined

    const speedUp = onSignIn && budget.linkedMultiplier

    const offer = speedUp && !compact && !slowdown

    return <div className="click-budget-dock panel">
        <div className="click-budget-shell">
            <div
                ref={root}
                className="click-budget"
                role="meter"
                aria-valuemin={0}
                aria-valuenow={whole}
                aria-valuemax={budget.capacity}
                aria-label="Clicks left before the server slows you down"
                style={{"--click-budget-capacity": budget.capacity} as React.CSSProperties}>
                <div className="click-budget-count">
                    <span ref={count} className="click-budget-number">{whole}</span>
                    <span className="click-budget-unit">left</span>
                </div>

                {pips > 0
                    ? <div className="click-budget-pips">
                        {Array.from({length: pips}, (_, index) =>
                            <span
                                key={index}
                                className="click-budget-pip"
                                style={{"--click-budget-index": index} as React.CSSProperties}/>,
                        )}
                    </div>
                    : <div className="click-budget-bar"/>}
            </div>

            <div className="click-budget-line">
                {slowdown && <span className="chip click-budget-toll"
                                   role="img"
                                   aria-label={`Refills ${factor(slowdown)}× slower`}>
                    <HourglassIcon/>
                    <span aria-hidden="true">{compact ? `${factor(slowdown)}×` : `${factor(slowdown)}× slower`}</span>
                </span>}
                {slow(budget) && <span ref={wait} className="click-budget-next">{waitText(budget, now())}</span>}
                {budget.sharedWith && <span className="click-budget-shared"
                                            role="img"
                                            aria-label={SHARED_WITH[budget.sharedWith]}
                                            title={SHARED_WITH[budget.sharedWith]}>
                    <UsersIcon size={15}/>
                </span>}
                {offer && <button type="button"
                                  className="button button-mini button-action click-budget-sign-in"
                                  onClick={onSignIn}>
                    <BoltIcon/>
                    <span>{offerText(budget.sharedWith, speedUp)}</span>
                </button>}
            </div>

            {onToggleOpen && <button type="button"
                                     className="click-budget-open"
                                     aria-label="Your clicks"
                                     aria-expanded={open}
                                     onClick={onToggleOpen}/>}
        </div>

        {children}

        {open && !compact && popover && onToggleOpen && <Popover onClose={onToggleOpen}>{popover}</Popover>}
    </div>
}

function Popover({children, onClose}: {children: ReactNode, onClose: () => void}) {
    useEscape(onClose)
    return <div className="click-budget-popover panel">{children}</div>
}

function slow(budget: ClickBudget): boolean {
    return budget.perSecond > 0 && 1 / budget.perSecond >= COUNTDOWN_FROM_S
}

function waitText(budget: ClickBudget, at: number): string {
    const left = secondsToOneMore(budget, at)
    return left === undefined ? "" : `+1 in ${Math.ceil(left)}s`
}

export function BoltIcon() {
    return <svg className="click-budget-sign-in-icon" width="14" height="14" viewBox="0 0 24 24" fill="currentColor"
                aria-hidden="true">
        <path d="M13 2 4 14h7l-1 8 9-12h-7z"/>
    </svg>
}
