import {useEffect, useRef} from "react"
import {Countries} from "../../domain/countries.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import {DiscordIcon} from "../components/icons.tsx"
import {usePing} from "./usePing.ts"
import "../components/BombNews.css"
import "./UnderAttack.css"

export const UNDER_ATTACK_MS = 8000

export type UnderAttackProps = {
    flag: string
    leave: () => Promise<void>
    lowered?: boolean
    onDone: () => void
}

export default function UnderAttack({flag, leave, lowered, onDone}: UnderAttackProps) {
    const ping = usePing(leave)
    const name = Countries.get(flag)?.name ?? flag

    const done = useRef(onDone)
    useEffect(() => {
        done.current = onDone
    }, [onDone])

    useEffect(() => {
        const timer = setTimeout(() => done.current(), UNDER_ATTACK_MS)
        return () => clearTimeout(timer)
    }, [])

    return <div className={`bomb-news under-attack${lowered ? " bomb-news--lowered" : ""}`} role="status" aria-live="polite">
        <div className="bomb-news-line under-attack-line panel" style={{animationDuration: `${UNDER_ATTACK_MS}ms`}}>
            <span className="under-attack-text">
                <CountryFlag code={flag}/>
                <span><strong>{name}</strong> is under attack</span>
            </span>
            <button type="button"
                    className="button button-mini under-attack-ping"
                    aria-label={ping.name}
                    aria-busy={ping.busy}
                    disabled={ping.busy}
                    onClick={ping.press}>
                <DiscordIcon size={18}/>
                {ping.label}
            </button>
        </div>
    </div>
}
