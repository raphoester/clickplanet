import {describe, expect, it} from "vitest"
import {NoteKind} from "../gen/grpc/support/v1/admin_pb.ts"
import {PageContext, ReportReason} from "../gen/grpc/support/v1/support_pb.ts"
import {deviceLines, kindLabel, reasonLabel} from "./labels.ts"

describe("deviceLines", () => {
    it("says everything the page said of itself, and the browser", () => {
        const context = new PageContext({
            build: "play-abc123.js",
            viewportWidth: 390,
            viewportHeight: 844,
            pixelRatio: 3,
            connectionDown: true,
            gpuVendor: "Apple",
            gpuRenderer: "Apple GPU",
            webglVersion: "WebGL 2.0",
            maxTextureSize: 16384,
            contextLost: true,
            cores: 6,
            memoryGb: 4,
        })

        expect(deviceLines(context, "Mozilla/5.0")).toEqual([
            {label: "Build", value: "play-abc123.js"},
            {label: "Viewport", value: "390×844 at 3x"},
            {label: "Connection", value: "down"},
            {label: "GPU", value: "Apple · Apple GPU"},
            {label: "WebGL", value: "WebGL 2.0, textures up to 16384px"},
            {label: "WebGL context", value: "lost"},
            {label: "Cores", value: "6"},
            {label: "Memory", value: "4 GB"},
            {label: "Browser", value: "Mozilla/5.0"},
        ])
    })

    it("leaves out what a note sent before the context was kept does not have", () => {
        expect(deviceLines(new PageContext(), "")).toEqual([])
        expect(deviceLines(undefined, "Mozilla/5.0")).toEqual([{label: "Browser", value: "Mozilla/5.0"}])
    })
})

describe("labels", () => {
    it("names every kind and reason the proto has", () => {
        expect(kindLabel(NoteKind.SUGGESTION)).toBe("Idea")
        expect(kindLabel(NoteKind.BUG)).toBe("Bug")
        expect(reasonLabel(ReportReason.CHAT_ABUSE)).toBe("Chat abuse")
    })

    it("names a value added to the proto since this build by its number", () => {
        expect(kindLabel(9 as NoteKind)).toBe("Kind 9")
        expect(reasonLabel(9 as ReportReason)).toBe("Reason 9")
    })
})
