import {useEffect, useRef} from 'react'
import {BonusReward, describeReward} from '../../domain/bonus.ts'
import './BonusAward.css'

/**
 * How long the announcement is on screen, matched to the `bonus-award` keyframes
 * in the CSS — it shouts, holds long enough to be read, then shrinks away
 * towards the meter, which is where the bonus lives for the rest of its life.
 */
export const AWARD_MS = 1900

export type BonusAwardProps = {
    reward: BonusReward
    onDone: () => void
}

/**
 * The moment a box is caught: big in the middle of the screen, then gone.
 *
 * It does not linger, and it is deliberately not the thing that says a bonus is
 * *running* — that is the click meter's job, because the meter is what the
 * allowance is actually read off. Two places both claiming to say how long is
 * left is two places that can disagree.
 *
 * Nothing here is clickable: it is over the globe for under two seconds, and a
 * player mid-click must not have the planet taken away from under the cursor.
 */
export default function BonusAward({reward, onDone}: BonusAwardProps) {
    const {title, detail} = describeReward(reward)

    // The callback is held in a ref rather than depended on, so that **only a
    // new reward** restarts the countdown.
    //
    // A caller that rebuilds `onDone` every render is ordinary, and this one
    // does: it sits beside a leaderboard that republishes twice a second. With
    // `onDone` in the dependencies the effect tore down and restarted the timer
    // on every one of those renders, so the announcement never reached its own
    // deadline and stayed up for good.
    const done = useRef(onDone)
    useEffect(() => {
        done.current = onDone
    }, [onDone])

    useEffect(() => {
        const timer = setTimeout(() => done.current(), AWARD_MS)
        return () => clearTimeout(timer)
    }, [reward])

    return <div className="bonus-award" role="status" aria-live="polite">
        <div className="bonus-award-card">
            <span className="bonus-award-box" aria-hidden="true">?</span>
            <strong className="bonus-award-title">{title}</strong>
            <span className="bonus-award-detail">{detail}</span>
        </div>
    </div>
}
