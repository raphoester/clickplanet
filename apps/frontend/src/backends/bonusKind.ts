import {BonusKind} from "../gen/grpc/planet/v1/planet_pb.ts"
import {ChargeKind} from "../domain/bonus.ts"

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
        default:
            return undefined
    }
}
