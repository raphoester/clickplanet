import {t} from "../../i18n/page.ts"
import "./Level.css"

export default function LevelChip({level, className}: {level: number, className?: string}) {
    return <span className={className ? `level-chip ${className}` : "level-chip"}
                 role="img"
                 aria-label={t("level.level", {level})}>
        <span aria-hidden="true">{t("level.short", {level})}</span>
    </span>
}
