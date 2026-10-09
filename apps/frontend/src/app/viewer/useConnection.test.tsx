// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import {ConnectionHealth, ConnectionSource, DOWN_AFTER_MISSES} from "../../backends/connection.ts"
import ConnectionLost from "../components/ConnectionLost.tsx"
import {useConnection} from "./useConnection.ts"

afterEach(cleanup)

function Probe({source}: {source: ConnectionSource | undefined}) {
    return useConnection(source) === "down" ? <ConnectionLost/> : null
}

describe("useConnection", () => {
    it("raises the card while the server cannot be reached, and drops it on the first answer", () => {
        const health = new ConnectionHealth()
        render(<Probe source={health}/>)
        expect(screen.queryByRole("alert")).toBeNull()

        act(() => {
            for (let i = 0; i < DOWN_AFTER_MISSES; i++) health.missed()
        })
        expect(screen.getByRole("alert").textContent).toContain("Connection lost")

        act(() => health.reached())
        expect(screen.queryByRole("alert")).toBeNull()
    })

    it("stays up with nothing to watch", () => {
        render(<Probe source={undefined}/>)

        expect(screen.queryByRole("alert")).toBeNull()
    })
})
