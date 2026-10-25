import {LONG_CALL_MS} from "../api.ts"
import {PlanetAdmin} from "../clients.ts"
import {ChargeFill, FillSource, GetChargeFillsResponse} from "../gen/grpc/planet/v1/admin_pb.ts"

export function runningOf(fills: GetChargeFillsResponse): boolean {
    return fills.fills.some(fill => fill.finishedAt === undefined)
}

export function stateOf(fill: ChargeFill): string {
    return fill.finishedAt === undefined ? "Running" : "Done"
}

export function fillSourceOf(source: FillSource): string {
    switch (source) {
        case FillSource.INTERNAL:
            return "Seasons"
        case FillSource.OPERATOR:
            return "An operator"
        case FillSource.UNSPECIFIED:
            return "Unknown"
    }
}

// A request of its own, so a press sent twice fills once.
export async function fillCharges(planet: PlanetAdmin, requestId: string = crypto.randomUUID()): Promise<void> {
    await planet.fillCharges({requestId}, {timeoutMs: LONG_CALL_MS})
}
