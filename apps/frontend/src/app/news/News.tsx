import {useEffect, useState} from "react"
import {Fortified} from "../../domain/fortify.ts"
import {Moved} from "../../domain/missionProgress.ts"
import UnderAttack from "../alerts/UnderAttack.tsx"
import MissionProgress from "../missions/MissionProgress.tsx"
import BombNews from "./BombNews.tsx"
import FortifyCard from "../components/FortifyCard.tsx"
import FortifyNews from "./FortifyNews.tsx"
import WelcomeNews from "./WelcomeNews.tsx"
import {lastFortified, newestNews, News as Told, useNews} from "./newsStore.ts"
import {useFortifyGuide} from "../components/useFortifyGuide.ts"
import {useCountry} from "../countryStore.ts"

// After the wave has run over the land.
export const FORTIFY_CARD_DELAY_MS = 1600

export type NewsProps = {
    connect?: () => Promise<void>
    goThere: () => void
    progress?: Moved
}

export default function News({connect, goThere, progress}: NewsProps) {
    const flag = useCountry((store) => store.country.code)
    const newest = useNews(newestNews)
    const fortified = useNews(lastFortified)
    const dismiss = useNews((news) => news.dismiss)

    const fortifyGuide = useFortifyGuide()
    const [card, setCard] = useState<Fortified | undefined>()
    useEffect(() => {
        if (!fortified || fortifyGuide.seen) return
        const timer = setTimeout(() => setCard(fortified.fortified), FORTIFY_CARD_DELAY_MS)
        return () => clearTimeout(timer)
    }, [fortified, fortifyGuide.seen])

    return <>
        {newest
            ? <Line news={newest} flag={flag} connect={connect} goThere={goThere} onDone={() => dismiss(newest)}/>
            : progress && <MissionProgress key={progress.id} moved={progress}/>}
        {card && <FortifyCard fortified={card} onDone={() => {
            fortifyGuide.markSeen()
            setCard(undefined)
        }}/>}
    </>
}

function Line({news, flag, connect, goThere, onDone}: {news: Told, flag: string, onDone: () => void} & Omit<NewsProps, "progress">) {
    switch (news.kind) {
        case "bomb":
            return <BombNews key={news.id} drop={news.drop} land={news.land} onDone={onDone}/>
        case "fortify":
            return <FortifyNews key={news.id} fortified={news.fortified} onDone={onDone}/>
        case "attack":
            return <UnderAttack key={news.id} flag={flag} connect={connect} goThere={goThere} onDone={onDone}/>
        case "welcome":
            return <WelcomeNews key={news.id} line={news.line} onDone={onDone}/>
    }
}
