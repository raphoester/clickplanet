import {useCallback, useEffect, useState} from "react";
import {GifChoice, GifQuery, GifSearch} from "../../backends/gifs.ts";

export const TYPING_PAUSE_MS = 350

export type GifSearchStatus = 'loading' | 'ready' | 'failed'

type Asked = {
    words: string
    page: number
}

type Found = {
    choices: GifChoice[]
    more: boolean
    status: GifSearchStatus
}

// A search starts once the typing pauses; each page adds to what the earlier ones found.
export function useGifSearch(gifs: GifSearch, typed: string, locale: string, customer: string) {
    const [asked, setAsked] = useState<Asked>({words: typed.trim(), page: 1})
    const [found, setFound] = useState<Found>({choices: [], more: false, status: 'loading'})

    useEffect(() => {
        const words = typed.trim()
        const timer = setTimeout(() => setAsked(current => current.words === words ? current : {words, page: 1}),
            TYPING_PAUSE_MS)
        return () => clearTimeout(timer)
    }, [typed])

    useEffect(() => {
        const abort = new AbortController()
        const first = asked.page === 1
        setFound(current => first ? {choices: [], more: false, status: 'loading'} : {...current, status: 'loading'})

        gifs.find({...asked, locale, customer}, abort.signal)
            .then(page => {
                if (abort.signal.aborted) return
                setFound(current => ({
                    choices: first ? page.choices : withNew(current.choices, page.choices),
                    more: page.more,
                    status: 'ready',
                }))
            })
            .catch(e => {
                if (abort.signal.aborted) return
                console.error("The GIFs could not be loaded", e)
                setFound(current => ({...current, status: 'failed'}))
            })

        return () => abort.abort()
    }, [gifs, asked, locale, customer])

    const {status, more} = found
    const loadMore = useCallback(() => {
        if (status === 'ready' && more) setAsked(current => ({...current, page: current.page + 1}))
    }, [status, more])

    const retry = useCallback(() => setAsked(current => ({...current})), [])

    const query: GifQuery = {...asked, locale, customer}
    return {...found, query, loadMore, retry}
}

// A page can hold one an earlier page did, once trending moved on.
function withNew(shown: GifChoice[], incoming: GifChoice[]): GifChoice[] {
    const slugs = new Set(shown.map(choice => choice.slug))
    return [...shown, ...incoming.filter(choice => !slugs.has(choice.slug))]
}
