import {useEffect, useRef} from "react"
import CountryFlag from "../components/CountryFlag.tsx"
import {DiscordIcon, TargetIcon} from "../components/icons.tsx"
import {pingFailureOf, pingNameOf} from "./alertsMessages.ts"
import {usePing} from "./usePing.ts"
import {t, translator} from "../../i18n/page.ts"
import {rich} from "../../i18n/rich.tsx"
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
    const name = translator().country(flag)

    const done = useRef(onDone)
    useEffect(() => {
        done.current = onDone
    }, [onDone])

    useEffect(() => {
        const timer = setTimeout(() => done.current(), UNDER_ATTACK_MS)
        return () => clearTimeout(timer)
    }, [])

    return <div className="bomb-news under-attack" role="status" aria-live="polite">
        <div className="bomb-news-line under-attack-line panel" style={{animationDuration: `${UNDER_ATTACK_MS}ms`}}>
            <span className="under-attack-text">
                <CountryFlag code={flag}/>
                <span>{rich("alerts.underAttack", {flag: <strong>{name}</strong>})}</span>
            </span>
            <button type="button" className="button button-mini button-secondary under-attack-go" onClick={goThere}>
                <TargetIcon size={18}/>
                {t("alerts.goThere")}
            </button>
            {connect && <PingButton name={name} connect={connect}/>}
        </div>
    </div>
}

function PingButton({name, connect}: {name: string, connect: () => Promise<void>}) {
    const ping = usePing(connect)
    const label = ping.failure ? pingFailureOf(ping.failure) : t("alerts.getPinged")

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
