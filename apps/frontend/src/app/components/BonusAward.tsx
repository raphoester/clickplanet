import {ReactNode, useEffect, useRef} from 'react'
import {BonusReward, describeReward} from '../../domain/bonus.ts'
import BonusIcon from './BonusIcon.tsx'
import {useEscape} from './useDialog.ts'
import './BonusAward.css'

export const AWARD_MS = 2900

export const DISMISS_GRACE_MS = 400

export type BonusAwardProps = {
    reward: BonusReward
    onDone: () => void
    kept?: boolean
}

export default function BonusAward({reward, onDone, kept = false}: BonusAwardProps) {
    return kept
        ? <KeptAward reward={reward} onDone={onDone}/>
        : <PassingAward reward={reward} onDone={onDone}/>
}

function PassingAward({reward, onDone}: BonusAwardProps) {
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

    return <AwardCard reward={reward}/>
}

function KeptAward({reward, onDone}: BonusAwardProps) {
    useEscape(onDone)

    return <AwardCard reward={reward} kept>
        <button type="button" className="button button-gold bonus-award-done" onClick={onDone}>Got it</button>
    </AwardCard>
}

function AwardCard({reward, kept = false, children}: {reward: BonusReward, kept?: boolean, children?: ReactNode}) {
    const {title, detail} = describeReward(reward)
    const className = ["bonus-award", `bonus-award--${reward.kind}`, kept && "bonus-award--kept"].filter(Boolean).join(" ")

    return <div className={className} role="status" aria-live="polite">
        <div className="bonus-award-card panel">
            <span className="bonus-award-box" aria-hidden="true"><BonusIcon kind={reward.kind}/></span>
            <strong className="bonus-award-title">{title}</strong>
            {detail && <span className="bonus-award-detail">{detail}</span>}
            {children}
        </div>
    </div>
}
