// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import App from "./App.tsx"
import type {Backends} from "./play/PlayScreen.tsx"
import {NO_FILTER} from "../domain/wordFilter.ts"

vi.mock("./play/PlayScreen.tsx", () => ({default: () => <div data-testid="play-screen"/>}))

const backends = {
    backends: {} as Backends,
    words: NO_FILTER,
}

afterEach(() => {
    cleanup()
})

describe("App", () => {
    it("renders the globe", () => {
        render(<App {...backends}/>)
        expect(screen.getByTestId("play-screen")).toBeDefined()
    })
})
