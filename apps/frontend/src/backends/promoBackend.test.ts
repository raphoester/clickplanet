import {describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {GetTeaserResponse, TeaserCard} from "../gen/grpc/promo/v1/promo_pb.ts"
import {ConnectPromoBackend} from "./promoBackend.ts"

const backendWith = (getTeaser: unknown) => new ConnectPromoBackend({getTeaser} as never)

describe("ConnectPromoBackend", () => {
    it("reads a battle's cards by its id", async () => {
        const getTeaser = vi.fn(async () => new GetTeaserResponse({cards: [
            new TeaserCard({backgroundUrl: "https://media.test/promo/backgrounds/war", headline: "Clicks refill 3× faster", line: "Every country at once."}),
        ]}))

        expect(await backendWith(getTeaser).teaser("tonight")).toEqual([
            {backgroundUrl: "https://media.test/promo/backgrounds/war", headline: "Clicks refill 3× faster", line: "Every country at once."},
        ])
        expect(getTeaser).toHaveBeenCalledWith({battleId: "tonight"})
    })

    it("reads no cards for a battle the schedule no longer holds, or a server without promo", async () => {
        for (const code of [Code.NotFound, Code.Unimplemented]) {
            const getTeaser = vi.fn(async () => {
                throw new ConnectError("no", code)
            })
            expect(await backendWith(getTeaser).teaser("gone")).toEqual([])
        }
    })
})
