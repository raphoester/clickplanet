import {BonusReward} from "../../domain/bonus.ts"
import {t} from "../../i18n/page.ts"

export function describeReward(reward: BonusReward): {
    title: string
    detail?: string
} {
    switch (reward.kind) {
        case "refill":
            return {title: t("bonus.refill"), detail: t("reward.refill.detail")}
        case "spreadClicks":
            return {title: t("reward.spreadClicks", {clicks: reward.clicks}), detail: t("bonus.spread.switchOn")}
        case "bomb":
            return {title: t("bonus.bomb"), detail: t("reward.bomb.detail")}
        case "encloseClicks":
            return {
                title: t("reward.encloseClicks", {shapes: reward.shapes}),
                detail: t("reward.encloseClicks.detail", {maxTiles: reward.maxTiles}),
            }
        case "shields":
            return {title: t("reward.shields", {shields: reward.shields}), detail: t("reward.shields.detail")}
    }
}
