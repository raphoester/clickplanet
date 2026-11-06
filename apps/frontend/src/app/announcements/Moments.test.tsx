// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {FakeClipsBackend} from "../../backends/fakeClipsBackend.ts"
import {FakePromoBackend} from "../../backends/fakePromoBackend.ts"
import {FlagMission} from "../../backends/missions.ts"
import {PlayerTitle} from "../../backends/progression.ts"
import {BATTLE_TEASED_STORAGE_KEY} from "../../domain/battleTeaser.ts"
import {TITLE_REVEAL} from "../../domain/titleReveal.ts"
import {acceptedClicks} from "../viewer/acceptedClicks.ts"
import {useAnnouncements} from "./announcementsStore.ts"
import Moments from "./Moments.tsx"

const raider: PlayerTitle = {id: "raider", name: "Raider", rank: {trackId: "conquest", trackName: "Conquest", number: 2, count: 5}}
const mission: FlagMission = {
    id: "flag-22",
    order: {kind: "breakShields", country: "pw", goal: 450, reached: 450},
    points: 2,
    done: true,
}

const clicks = acceptedClicks()
const moments = (shown: boolean) => <Moments shown={shown}
                                             guided={false}
                                             promo={new FakePromoBackend()}
                                             clips={new FakeClipsBackend()}
                                             onWear={vi.fn()}
                                             listenForClicks={clicks.listenForClicks}
                                             countryCode="fr"
                                             play={vi.fn()}/>

function queueTwo() {
    const {announce} = useAnnouncements.getState()
    announce({kind: "title", title: raider})
    announce({kind: "flagMission", mission, country: "fr"})
}

beforeEach(() => {
    vi.useFakeTimers({shouldAdvanceTime: true})
    window.localStorage.clear()
})

afterEach(() => {
    cleanup()
    useAnnouncements.setState({queue: []})
    vi.useRealTimers()
})

describe("Moments", () => {
    it("shows nothing before the map is loaded, then the first moment queued", () => {
        queueTwo()
        const {rerender} = render(moments(false))
        expect(screen.queryByRole("dialog")).toBeNull()

        rerender(moments(true))

        expect(screen.getByRole("dialog", {name: "New title! Raider"})).toBeDefined()
    })

    it("shows the next moment once the player closes the first", async () => {
        queueTwo()
        render(moments(true))

        act(() => vi.advanceTimersByTime(TITLE_REVEAL.ready * 1000))
        await userEvent.setup({advanceTimers: vi.advanceTimersByTime}).click(screen.getByRole("button", {name: "Close"}))

        expect(screen.getByRole("dialog", {name: "Congratulations! France"})).toBeDefined()
        expect(useAnnouncements.getState().queue).toHaveLength(1)
    })

    it("teases the next battle with its cards, and keeps that it did once closed", async () => {
        const promo = new FakePromoBackend()
        const startsAt = Date.now() + 3 * 60 * 60 * 1000
        const next = {entryId: "tonight", startsAt, endsAt: startsAt + 60 * 60 * 1000, description: "", times: 2, endsSeason: false, roundId: "tonight"}
        render(<Moments shown guided={false} nextBattle={next} promo={promo} clips={new FakeClipsBackend()} listenForClicks={clicks.listenForClicks}
                        countryCode="fr" play={vi.fn()} onWear={vi.fn()}/>)

        expect(await screen.findByRole("dialog", {name: "Battle"})).toBeDefined()
        act(() => vi.advanceTimersByTime(TITLE_REVEAL.ready * 1000))
        await userEvent.setup({advanceTimers: vi.advanceTimersByTime}).click(screen.getByRole("button", {name: "To the globe"}))

        expect(screen.queryByRole("dialog")).toBeNull()
        expect(JSON.parse(window.localStorage.getItem(BATTLE_TEASED_STORAGE_KEY) ?? "")).toMatchObject({key: "tonight"})
    })
})
