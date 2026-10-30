import {useCallback, useEffect, useRef, useState} from "react"
import {AdRefiller, AdRefillTooSoonError, BankFullError} from "../../backends/backend.ts"
import {RewardedAds} from "../../backends/crazyGames.ts"

// off: no ad on this page, the rule off, or none in this session. unavailable: none right now, ask again later.
export type AdRefillState =
    | {kind: "off"}
    | {kind: "blocked"}
    | {kind: "ready"}
    | {kind: "playing"}
    | {kind: "refilled"}
    | {kind: "waiting", until: number}
    | {kind: "unavailable"}

export type AdRefill = {
    state: AdRefillState
    watch: (countryId: string) => void
    // The refill was shown: the clock to the next one comes after it.
    seen: () => void
}

const OFF: AdRefillState = {kind: "off"}

// The server fills the bank, and only once per span: the ad is the client's word, and the client is not trusted.
export function useAdRefill(ads: RewardedAds | undefined, refiller: AdRefiller | undefined, on: boolean, onPlaying: (playing: boolean) => void): AdRefill {
    const [state, setState] = useState<AdRefillState>(OFF)
    const onPlayingRef = useRef(onPlaying)
    onPlayingRef.current = onPlaying

    const readNext = useCallback(async (from: AdRefiller) => {
        try {
            const until = await from.adRefillAt()
            setState(until > Date.now() ? {kind: "waiting", until} : {kind: "ready"})
        } catch (e) {
            console.error("could not read when the next ad refill is", e)
            setState(OFF)
        }
    }, [])

    useEffect(() => {
        if (!ads || !refiller || !on) {
            setState(OFF)
            return
        }
        let live = true
        void ads.blocked().then((blocked) => {
            if (!live) return
            if (blocked) setState({kind: "blocked"})
            else void readNext(refiller)
        })
        return () => {
            live = false
        }
    }, [ads, refiller, on, readNext])

    const until = state.kind === "waiting" ? state.until : undefined
    useEffect(() => {
        if (until === undefined) return
        const timer = setTimeout(() => setState({kind: "ready"}), Math.max(0, until - Date.now()))
        return () => clearTimeout(timer)
    }, [until])

    const watch = useCallback((countryId: string) => {
        if (!ads || !refiller || (state.kind !== "ready" && state.kind !== "unavailable")) return
        setState({kind: "playing"})

        void ads.watch(() => onPlayingRef.current(true)).then(async (outcome) => {
            onPlayingRef.current(false)
            switch (outcome) {
                case "watched":
                    break
                case "unavailable":
                    setState({kind: "unavailable"})
                    return
                case "blocked":
                    setState({kind: "blocked"})
                    return
                case "off":
                    setState(OFF)
                    return
            }

            try {
                await refiller.claimAdRefill(countryId)
                setState({kind: "refilled"})
            } catch (e) {
                if (e instanceof AdRefillTooSoonError) setState({kind: "waiting", until: e.availableAt})
                else if (e instanceof BankFullError) void readNext(refiller)
                else {
                    console.error("could not claim the ad refill", e)
                    setState({kind: "unavailable"})
                }
            }
        })
    }, [ads, refiller, state.kind, readNext])

    const seen = useCallback(() => {
        if (state.kind === "refilled" && refiller) void readNext(refiller)
    }, [state.kind, refiller, readNext])

    return {state, watch, seen}
}
