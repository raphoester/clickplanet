import {useEffect, useRef} from "react"
import {Countries} from "../../domain/countries.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import {quizShown, useQuiz} from "../quiz/quizStore.ts"
import {DiscordIcon, TargetIcon} from "../components/icons.tsx"
import {pingFailureOf, pingNameOf} from "./alertsMessages.ts"
import {usePing} from "./usePing.ts"
import "../news/BombNews.css"
import "./UnderAttack.css"

export const UNDER_ATTACK_MS = 8000

export type UnderAttackProps = {
    flag: string
    connect?: () => Promise<void>
    goThere: () => void
    onDone: () => void
}

export default function UnderAttack({flag, connect, goThere, onDone}: UnderAttackProps) {
    const name = Countries.get(flag)?.name ?? flag
    const lowered = useQuiz(quizShown)

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
            <button type="button" className="button button-mini button-secondary under-attack-go" onClick={goThere}>
                <TargetIcon size={18}/>
                Go there
            </button>
            {connect && <PingButton name={name} connect={connect}/>}
        </div>
    </div>
}

function PingButton({name, connect}: {name: string, connect: () => Promise<void>}) {
    const ping = usePing(connect)
    const label = ping.failure ? pingFailureOf(ping.failure) : "Get pinged"

    return <button type="button"
                   className="button button-mini under-attack-ping"
                   aria-label={ping.failure ? label : pingNameOf(name)}
                   aria-busy={ping.busy}
                   disabled={ping.busy}
                   onClick={ping.press}>
        <DiscordIcon size={18}/>
        {label}
    </button>
}
