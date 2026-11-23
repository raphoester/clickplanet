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
            return {
                title: reward.clicks > 0 ? t("reward.spreadClicks", {clicks: reward.clicks}) : t("bonus.spread"),
                detail: t("bonus.spread.switchOn"),
            }
        case "bomb":
            return {
                title: reward.bombs > 1 ? t("reward.bombs", {bombs: reward.bombs}) : t("bonus.bomb"),
                detail: t("reward.bomb.detail"),
            }
        case "encloseClicks":
            return {
                title: reward.shapes > 0 ? t("reward.encloseClicks", {shapes: reward.shapes}) : t("bonus.enclose"),
                detail: t("reward.encloseClicks.detail", {maxTiles: reward.maxTiles}),
            }
        case "shields":
            return {
                title: reward.shields > 0 ? t("reward.shields", {shields: reward.shields}) : t("bonus.shield"),
                detail: t("reward.shields.detail"),
            }
        case "virus":
            return {
                title: reward.viruses > 1 ? t("reward.viruses", {viruses: reward.viruses}) : t("bonus.virus"),
                detail: t("reward.virus.detail"),
            }
        case "mine":
            return {
                title: reward.mines > 0 ? t("reward.mines", {mines: reward.mines}) : t("bonus.mine"),
                detail: t("reward.mines.detail"),
            }
        case "rally":
            return {
                title: reward.rallies > 0 ? t("reward.rallies", {rallies: reward.rallies}) : t("bonus.rally"),
                detail: t("reward.rallies.detail"),
            }
        case "trap":
            return {
                title: reward.traps > 0 ? t("reward.traps", {traps: reward.traps}) : t("bonus.trap"),
                detail: t("reward.traps.detail"),
            }
    }
}
