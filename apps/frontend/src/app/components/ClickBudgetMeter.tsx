import {useEffect, useRef} from 'react'
import {ClickBudget, nextClickProgress, now, secondsToOneMore, tokensAt} from "../../backends/clickBudget.ts"
import {ActiveBonus, chargeLabels, Charges, describeReward, NO_CHARGES, secondsLeft} from "../../domain/bonus.ts"
import {describePrice, factor} from "../../domain/clickPrice.ts"
import "./ClickBudgetMeter.css"

export type ClickBudgetMeterProps = {
    budget?: ClickBudget
    /**
     * The triple currently running, if any. The meter is the one place that says
     * a bonus is live, because it is where the allowance is read — and once the
     * backend grants the boost, the fill rate speeds up on its own off the
     * server's policy, with nothing here to change.
     */
    bonus?: ActiveBonus
    /**
     * The charges held: a bomb, an enclose, a spread's clicks. Said under the
     * meter with no countdown, since none of them runs out while the player
     * plays — each lasts until it is spent.
     */
    charges?: Charges
    /** Whether the bomb held is aimed, so its button can say which way it goes. */
    bombArmed?: boolean
    /** Aims the bomb held, or puts it away. Absent, the bomb is only said. */
    onToggleBomb?: () => void
    /** The country selected, to say why its clicks refill slower. */
    countryName?: string
    /**
     * How many clicks the server has refused for the throttle. Each new one
     * shakes the meter and flashes it red — the only thing said about a refused
     * click, since this is where the player already looks for the allowance.
     */
    refusals?: number
    /**
     * Present for a guest the server offers sign-in to: the meter then says
     * what signing in is worth, under the pips, and this opens the way to it.
     * The meter is where a guest meets the wall, so it is where the offer is.
     */
    onSignIn?: () => void
}

/** Above this many, a row of pips is unreadable and it becomes one bar. */
const MAX_PIPS = 12

/** Below this, the player is close enough to the wall to be warned. */
const LOW_WATER = 3

/**
 * A click that takes this long or more to come back gets a countdown. At one a
 * second it would only ever say 1s; at one every 5s, a count stuck on 0 and a
 * bar that barely moves look broken without it.
 */
const COUNTDOWN_FROM_S = 1.5

/** Reduced motion steps the fill instead of gliding it. */
const STEP_MS = 250

/**
 * How many clicks the server will still take, and the next one arriving.
 *
 * The count comes from the server and nowhere else — see clickBudget.ts for why
 * a client-side bucket could not tell the truth here. What this adds is the
 * only part that is honestly local: replaying the refill between two readings,
 * so the pip fills smoothly rather than jumping once per answer.
 *
 * Everything about the shape is read off the server's own policy. The number of
 * pips *is* the burst, and the fill rate *is* the refill rate, so changing
 * either in the backend's config changes this with no frontend release.
 */
export default function ClickBudgetMeter({
    budget,
    bonus,
    charges = NO_CHARGES,
    bombArmed = false,
    onToggleBomb,
    countryName = "",
    refusals = 0,
    onSignIn,
}: ClickBudgetMeterProps) {
    const root = useRef<HTMLDivElement>(null)
    const count = useRef<HTMLSpanElement>(null)
    const countdown = useRef<HTMLSpanElement>(null)
    const wait = useRef<HTMLSpanElement>(null)

    useEffect(() => {
        const box = root.current
        if (!box || refusals === 0) return

        // Take the class off and put it back, with a layout read between, so a
        // refusal during the animation restarts it rather than being lost.
        box.classList.remove("click-budget-refused")
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
        let shownSecond = -1
        let shownWait = ""

        const bar = budget.capacity > MAX_PIPS

        const draw = () => {
            const at = now()
            const tokens = tokensAt(budget, at)
            const whole = Math.floor(tokens)

            // Written the same way the count is — only when the displayed value
            // changes, so a 60 second bonus costs 60 writes and not 3,600.
            if (bonus && countdown.current) {
                const left = secondsLeft(bonus, at)
                if (left !== shownSecond) {
                    shownSecond = left
                    countdown.current.textContent = `${left}s`
                }
            }

            // One write, and every pip works out its own share of it.
            box.style.setProperty("--click-budget-tokens", tokens.toFixed(3))

            // A bar of 60 moves a sixtieth per click, too little to see; the
            // strip under it fills once per click, as a pip would.
            if (bar) box.style.setProperty("--click-budget-next", nextClickProgress(budget, at).toFixed(3))

            if (wait.current) {
                const text = waitText(budget, at)
                if (text !== shownWait) {
                    shownWait = text
                    wait.current.textContent = text
                }
            }

            // The rest changes about once a second, so it is not written per
            // frame — this sits beside a WebGL scene that wants the main thread.
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
    }, [budget, bonus])

    // A backend that reports no allowance is one that enforces none here.
    if (!budget) return null

    const pips = budget.capacity <= MAX_PIPS ? budget.capacity : 0

    // The effect redraws this on its first frame; rendering the reading rather
    // than a placeholder is what stops the count flashing a wrong number first.
    const whole = Math.floor(tokensAt(budget, now()))

    const price = describePrice(budget.price, countryName)
    const className = ["click-budget", bonus && "click-budget-boosted", price && "click-budget-priced"]
        .filter(Boolean).join(" ")

    // Said only when the server says what it is worth: a number made up here could promise what it does not grant.
    const speedUp = onSignIn && budget.linkedMultiplier

    return <div className="click-budget-dock"><div
        ref={root}
        className={className}
        role="meter"
        aria-valuemin={0}
        aria-valuenow={whole}
        aria-valuemax={budget.capacity}
        aria-label="Clicks left before the server slows you down"
        style={{"--click-budget-capacity": budget.capacity} as React.CSSProperties}>

        {bonus && <span className="click-budget-bonus">
            <span className="click-budget-bonus-badge">{describeReward(bonus.reward).badge}</span>
            <span ref={countdown} className="click-budget-bonus-left">{secondsLeft(bonus, now())}s</span>
        </span>}

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

        {slow(budget) && <span ref={wait} className="click-budget-next">{waitText(budget, now())}</span>}

        {price && <div className="click-budget-toll">
            <span className="click-budget-toll-headline">{price.headline}</span>
            <span className="click-budget-toll-detail">{price.detail}</span>
        </div>}
    </div>

        <ChargesHeld charges={charges} bombArmed={bombArmed} onToggleBomb={onToggleBomb}/>

        {speedUp && <button type="button" className="click-budget-sign-in" onClick={onSignIn}>
            <BoltIcon/>
            <span>Sign in: clicks {factor(speedUp)}× faster</span>
        </button>}
    </div>
}

function slow(budget: ClickBudget): boolean {
    return budget.perSecond > 0 && 1 / budget.perSecond >= COUNTDOWN_FROM_S
}

/** When the next click is in hand, or nothing at a full bucket. */
function waitText(budget: ClickBudget, at: number): string {
    const left = secondsToOneMore(budget, at)
    return left === undefined ? "" : `+1 in ${Math.ceil(left)}s`
}

/**
 * One pill per charge held. The bomb's is a button: a bomb held for a day
 * cannot stay aimed for a day, since an aimed bomb turns every click into a
 * press that drops it, so the player takes it out and puts it away here.
 */
function ChargesHeld({charges, bombArmed, onToggleBomb}: {
    charges: Charges
    bombArmed: boolean
    onToggleBomb?: () => void
}) {
    const labels = chargeLabels(charges)
    if (labels.length === 0) return null

    return <div className="click-budget-charges" role="status" aria-label="Bonuses held">
        {labels.map(({kind, label}) => kind === "bomb" && onToggleBomb
            ? <button key={kind}
                      type="button"
                      className={`click-budget-charge click-budget-charge--bomb${bombArmed ? " click-budget-charge--armed" : ""}`}
                      aria-pressed={bombArmed}
                      title={bombArmed ? "Put the bomb away (Esc)" : "Aim the bomb, then hold on the planet to drop it"}
                      onClick={onToggleBomb}>
                <span aria-hidden="true">💣</span> {bombArmed ? "Aiming: hold to drop" : label}
            </button>
            : <span key={kind} className={`click-budget-charge click-budget-charge--${kind}`}>{label}</span>)}
    </div>
}

function BoltIcon() {
    return <svg className="click-budget-sign-in-icon" width="14" height="14" viewBox="0 0 24 24" fill="currentColor"
                aria-hidden="true">
        <path d="M13 2 4 14h7l-1 8 9-12h-7z"/>
    </svg>
}
