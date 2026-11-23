import {BonusKind} from "../gen/grpc/planet/v1/planet_pb.ts"
import {BonusReward, BonusRules, ChargeKind} from "../domain/bonus.ts"

export function chargeKindOf(kind: BonusKind): ChargeKind | undefined {
    switch (kind) {
        case BonusKind.REFILL:
            return "refill"
        case BonusKind.SPREAD_CLICKS:
            return "spreadClicks"
        case BonusKind.BOMB:
            return "bomb"
        case BonusKind.ENCLOSE_CLICKS:
            return "encloseClicks"
        case BonusKind.SHIELDS:
            return "shields"
        case BonusKind.VIRUS:
            return "virus"
        case BonusKind.MINE:
            return "mine"
        case BonusKind.RALLY:
            return "rally"
        case BonusKind.TRAP:
            return "trap"
        default:
            return undefined
    }
}

type RewardRules = Pick<BonusRules, "blastRadius" | "enclosureMaxTiles">

const NO_REWARD_RULES: RewardRules = {blastRadius: 0, enclosureMaxTiles: 0}

export function rewardOf(
    kind: BonusKind,
    {blastRadius, enclosureMaxTiles: maxTiles}: RewardRules = NO_REWARD_RULES,
    amount = 1,
): BonusReward | undefined {
    switch (kind) {
        case BonusKind.REFILL:
            return {kind: "refill", refills: amount}
        case BonusKind.SPREAD_CLICKS:
            return {kind: "spreadClicks", clicks: amount}
        case BonusKind.BOMB:
            return {kind: "bomb", radius: blastRadius, bombs: amount}
        case BonusKind.ENCLOSE_CLICKS:
            return {kind: "encloseClicks", shapes: amount, maxTiles}
        case BonusKind.SHIELDS:
            return {kind: "shields", shields: amount}
        case BonusKind.VIRUS:
            return {kind: "virus", viruses: amount}
        case BonusKind.MINE:
            return {kind: "mine", mines: amount}
        case BonusKind.RALLY:
            return {kind: "rally", rallies: amount}
        case BonusKind.TRAP:
            return {kind: "trap", traps: amount}
        default:
            return undefined
    }
}
