import {useEffect, useRef} from 'react'
import {BonusReward, describeReward} from '../../domain/bonus.ts'
import BonusIcon from './BonusIcon.tsx'
import './BonusAward.css'

export const AWARD_MS = 2900

export const DISMISS_GRACE_MS = 400

export type BonusAwardProps = {
    reward: BonusReward
    onDone: () => void
}

export default function BonusAward({reward, onDone}: BonusAwardProps) {
    const {title, detail} = describeReward(reward)

    const done = useRef(onDone)
    useEffect(() => {
        done.current = onDone
    }, [onDone])

    useEffect(() => {
        let armed = false
        const grace = setTimeout(() => armed = true, DISMISS_GRACE_MS)
        const timer = setTimeout(() => done.current(), AWARD_MS)
        const dismiss = () => {
            if (armed) done.current()
        }
        window.addEventListener("pointerdown", dismiss, true)
        return () => {
            clearTimeout(grace)
            clearTimeout(timer)
            window.removeEventListener("pointerdown", dismiss, true)
        }
    }, [reward])

    return <div className={`bonus-award bonus-award--${reward.kind}`} role="status" aria-live="polite">
        <div className="bonus-award-card panel">
            <span className="bonus-award-box" aria-hidden="true"><BonusIcon kind={reward.kind}/></span>
            <strong className="bonus-award-title">{title}</strong>
            {detail && <span className="bonus-award-detail">{detail}</span>}
        </div>
    </div>
}
