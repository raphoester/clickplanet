import {useEffect, useRef} from 'react'
import {ClickBudget, now, tokensAt} from "../../backends/clickBudget.ts"
import {ActiveBonus, describeReward, secondsLeft} from "../../domain/bonus.ts"
import {describePrice} from "../../domain/clickPrice.ts"
import "./ClickBudgetMeter.css"

export type ClickBudgetMeterProps = {
    budget?: ClickBudget
    /**
     * The bonus currently running, if any. The meter is the one place that says
     * a bonus is live, because it is where the allowance is read — and once the
     * backend grants the boost, the pips and the fill rate widen on their own
     * off the server's policy, with nothing here to change.
     */
    bonus?: ActiveBonus
    /** The country clicks are priced for, to say why the meter is narrower. */
    countryName?: string
    /**
     * How many clicks the server has refused for the throttle. Each new one
     * shakes the meter and flashes it red — the only thing said about a refused
     * click, since this is where the player already looks for the allowance.
     */
    refusals?: number
}

/** Above this many, a row of pips is unreadable and it becomes one bar. */
const MAX_PIPS = 12

/** Below this, the player is close enough to the wall to be warned. */
const LOW_WATER = 3

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
export default function ClickBudgetMeter({budget, bonus, countryName = "", refusals = 0}: ClickBudgetMeterProps) {
    const root = useRef<HTMLDivElement>(null)
    const count = useRef<HTMLSpanElement>(null)
    const countdown = useRef<HTMLSpanElement>(null)

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

    return <div
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

        {price && <div className="click-budget-toll">
            <span className="click-budget-toll-headline">{price.headline}</span>
            <span className="click-budget-toll-detail">{price.detail}</span>
        </div>}
    </div>
}
