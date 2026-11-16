import {BonusReward} from "../../domain/bonus.ts"
import {t} from "../../i18n/page.ts"

export function describeReward(reward: BonusReward): {
    title: string
    detail?: string
} {
    switch (reward.kind) {
        case "refill":
            return {
                title: reward.refills > 1 ? t("reward.refills", {refills: reward.refills}) : t("bonus.refill"),
                detail: t("reward.refill.detail"),
            }
        case "spreadClicks":
            return {title: t("reward.spreadClicks", {clicks: reward.clicks}), detail: t("bonus.spread.switchOn")}
        case "bomb":
            return {
                title: reward.bombs > 1 ? t("reward.bombs", {bombs: reward.bombs}) : t("bonus.bomb"),
                detail: t("reward.bomb.detail"),
            }
        case "encloseClicks":
            return {
                title: t("reward.encloseClicks", {shapes: reward.shapes}),
                detail: t("reward.encloseClicks.detail", {maxTiles: reward.maxTiles}),
            }
        case "shields":
            return {title: t("reward.shields", {shields: reward.shields}), detail: t("reward.shields.detail")}
        case "virus":
            return {
                title: reward.viruses > 1 ? t("reward.viruses", {viruses: reward.viruses}) : t("bonus.virus"),
                detail: t("reward.virus.detail"),
            }
        case "mine":
            return {title: t("reward.mines", {mines: reward.mines}), detail: t("reward.mines.detail")}
        case "rally":
            return {title: t("reward.rallies", {rallies: reward.rallies}), detail: t("reward.rallies.detail")}
    }
}
