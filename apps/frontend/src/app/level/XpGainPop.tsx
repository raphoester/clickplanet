import {XpSource} from "../../backends/progression.ts"
import {XpPop} from "../../domain/xpPops.ts"
import {t, translator} from "../../i18n/page.ts"
import {Key} from "../../i18n/translator.ts"
import "./Level.css"

const WHY: Record<XpSource, Key> = {
    tilesTaken: "level.why.tilesTaken",
    daysPlayed: "level.why.daysPlayed",
    missionsDone: "level.why.missionsDone",
    bombsDropped: "level.why.bombsDropped",
    boxesCaught: "level.why.boxesCaught",
    rightAnswers: "level.why.rightAnswers",
    tilesFortified: "level.why.tilesFortified",
}

// A gain on the level's line, in the XP's place: how much and why. The amount bumps at each gain added to it, and only
// a gain that is not a take is read out, as a take comes at every click.
export default function XpGainPop({pop}: {pop: XpPop}) {
    const amount = t("level.gained", {xp: translator().number(pop.xp)})
    const why = t(WHY[pop.source])
    return <span className="xp-gain">
        <span key={pop.bumps} className="xp-gain-amount">{amount}</span>
        <span className="xp-gain-why">{why}</span>
        {pop.source !== "tilesTaken" && <span className="sr-only" role="status">{`${amount} ${why}`}</span>}
    </span>
}
