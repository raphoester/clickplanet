import {CSSProperties} from "react"
import {PlayerLevel} from "../../backends/progression.ts"
import {barOf} from "../../domain/level.ts"
import {t, translator} from "../../i18n/page.ts"
import "./Level.css"

export default function XpBar({level, className}: {level: PlayerLevel, className?: string}) {
    const bar = barOf(level)
    const words = translator()
    return <span className={className ? `xp-bar ${className}` : "xp-bar"}
                 role="meter"
                 aria-label={t("level.xpBar", {level: level.level + 1})}
                 aria-valuemin={0}
                 aria-valuemax={bar.span}
                 aria-valuenow={bar.gained}
                 aria-valuetext={t("level.xp", {gained: words.number(bar.gained), span: words.number(bar.span)})}
                 style={{"--xp-fill": bar.fill} as CSSProperties}>
        <span className="xp-bar-fill"/>
    </span>
}
