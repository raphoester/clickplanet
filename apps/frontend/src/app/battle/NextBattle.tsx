import {BattleToCome} from "../../backends/turn.ts"
import {teaserKeyOf} from "../../domain/battleTeaser.ts"
import {t} from "../../i18n/page.ts"
import {useAnnouncements} from "../announcements/announcementsStore.ts"
import {SwordsIcon} from "../components/icons.tsx"
import {useNow} from "../season/useNow.ts"
import {battleWhenLine} from "./battleLines.ts"
import {useBattleCards} from "./teaserCardsStore.ts"
import "./Battle.css"

// The next battle, in the season chip's details: when it plays in the player's time, what it is worth, the operator's
// words, and its rules a press away.
export function NextBattleDetails({battle}: {battle: BattleToCome}) {
    const now = useNow()

    return <div className="finale-details next-battle-details">
        <p className="finale-details-head">
            <SwordsIcon/>
            <span className="finale-details-name">{t("battle.next")}</span>
        </p>
        <p className="finale-details-when">{battleWhenLine(battle, now)}</p>
        {battle.description && <p className="finale-details-words">{battle.description}</p>}
        <BattleRulesButton battle={battle}/>
    </div>
}

// The battle's rule set's cards, a press away: its teaser, with no entrance, through the announcements queue. None
// while promo has no card for it, as a battle with none is not teased.
export function BattleRulesButton({battle}: {battle: BattleToCome | undefined}) {
    const cards = useBattleCards(battle && teaserKeyOf(battle))
    const announce = useAnnouncements((announcements) => announcements.announce)

    if (!battle || !cards || cards.length === 0) return null
    return <button type="button"
                   className="button button-mini button-action battle-rules-button"
                   onClick={() => announce({kind: "teaser", battle, cards, asked: true})}>
        {t("battle.seeTheRules")}
    </button>
}
