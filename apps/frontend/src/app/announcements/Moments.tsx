import {useEffect} from "react"
import {ClipsBackend} from "../../backends/clips.ts"
import {ClosedRound} from "../../backends/standings.ts"
import {announcementKeyOf} from "../../domain/announcements.ts"
import {Battle, Finale} from "../../domain/battle.ts"
import {briefDescriptionOf} from "../../domain/battleBrief.ts"
import {revealFollowed} from "../../domain/roundReveal.ts"
import BattleDebrief from "../battle/BattleDebrief.tsx"
import BattleReveal from "../battle/BattleReveal.tsx"
import FinaleTeaser from "../battle/FinaleTeaser.tsx"
import {useBattleReveal} from "../battle/useBattleReveal.ts"
import {useFinaleTeaser} from "../battle/useFinaleTeaser.ts"
import {useSeenBattle} from "../battle/useSeenBattle.ts"
import ClipNotice from "../clips/ClipNotice.tsx"
import {useClipNotice} from "../clips/useClipNotice.ts"
import FlagMissionDone from "../missions/FlagMissionDone.tsx"
import RoundReveal from "../season/RoundReveal.tsx"
import {useRoundReveal} from "../season/useRoundReveal.ts"
import {PlaySound} from "../sound/soundPlayer.ts"
import TitleUnlocked from "../titles/TitleUnlocked.tsx"
import {ListenForClicks} from "../viewer/acceptedClicks.ts"
import {useAnnouncements} from "./announcementsStore.ts"

export type MomentsProps = {
    shown: boolean
    battle?: Battle
    closed?: ClosedRound
    finale?: Finale
    listenForClicks: ListenForClicks
    clips?: ClipsBackend
    clipsAccount?: string
    countryCode: string
    accountId?: string
    play: PlaySound
    onWear?: (id: string) => Promise<unknown>
}

export default function Moments(props: MomentsProps) {
    const current = useAnnouncements((announcements) => announcements.queue[0])
    const announce = useAnnouncements((announcements) => announcements.announce)
    const dismiss = useAnnouncements((announcements) => announcements.dismiss)

    const seenBattle = useSeenBattle(props.battle)
    const reveal = useRoundReveal(props.closed, seenBattle)
    const battleReveal = useBattleReveal(props.battle, props.closed)
    const finaleTeaser = useFinaleTeaser(props.finale, Date.now(), props.listenForClicks)
    const clipNotice = useClipNotice(props.clips, props.clipsAccount)

    useEffect(() => {
        for (const due of reveal.due) announce(due)
    }, [announce, reveal.due])
    useEffect(() => {
        if (battleReveal.battle) announce({kind: "battle", battle: battleReveal.battle})
    }, [announce, battleReveal.battle])
    useEffect(() => {
        if (finaleTeaser.finale) announce({kind: "finale", finale: finaleTeaser.finale})
    }, [announce, finaleTeaser.finale])
    useEffect(() => {
        if (clipNotice.appearance) announce({kind: "clip", appearance: clipNotice.appearance})
    }, [announce, clipNotice.appearance])

    if (!props.shown || !current) return null

    const key = announcementKeyOf(current)
    switch (current.kind) {
        case "title":
            return <TitleUnlocked key={key}
                                  title={current.title}
                                  play={props.play}
                                  onWear={props.onWear}
                                  onClose={dismiss}/>
        case "round":
        case "results":
            return <RoundReveal key={key}
                                closed={current.closed}
                                results={current.kind === "results"}
                                countryCode={props.countryCode}
                                play={props.play}
                                onClose={() => {
                                    reveal.dismiss(current)
                                    dismiss()
                                }}/>
        case "over":
        case "points":
        case "players":
            return <BattleDebrief key={key}
                                  screen={current.kind}
                                  closed={current.closed}
                                  description={briefDescriptionOf(seenBattle, current.closed)}
                                  countryCode={props.countryCode}
                                  accountId={props.accountId}
                                  more={revealFollowed(current, seenBattle)}
                                  play={props.play}
                                  onClose={() => {
                                      reveal.dismiss(current)
                                      dismiss()
                                  }}/>
        case "battle":
            return <BattleReveal key={key}
                                 battle={current.battle}
                                 play={props.play}
                                 onClose={() => {
                                     battleReveal.dismiss()
                                     dismiss()
                                 }}/>
        case "finale":
            return <FinaleTeaser key={key}
                                 finale={current.finale}
                                 play={props.play}
                                 onClose={() => {
                                     finaleTeaser.dismiss(current.finale)
                                     dismiss()
                                 }}/>
        case "clip":
            return <ClipNotice key={key}
                               appearance={current.appearance}
                               play={props.play}
                               onClose={() => {
                                   clipNotice.dismiss(current.appearance)
                                   dismiss()
                               }}/>
        case "flagMission":
            return <FlagMissionDone key={key}
                                    mission={current.mission}
                                    country={current.country}
                                    play={props.play}
                                    onClose={dismiss}/>
    }
}
