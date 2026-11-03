import {KeyboardEvent, useEffect, useRef, useState} from "react";
import {GifChoice, GifQuery, GifSearch} from "../../backends/gifs.ts";
import {CloseIcon, LockIcon} from "../components/icons.tsx";
import {useEscape} from "../components/useDialog.ts";
import {useGifSearch} from "./useGifSearch.ts";
import "./GifPicker.css"

export type GifOffer = {
    search: GifSearch
    locale: string
    customer: string
    // Set for a guest, who may not send one.
    onSignIn?: () => void
}

export type GifPickerProps = {
    offer: GifOffer
    onPick: (choice: GifChoice, query: GifQuery) => void
    onClose: () => void
}

export default function GifPicker({offer, onPick, onClose}: GifPickerProps) {
    useEscape(onClose)

    return <div className="gif-picker" role="group" aria-label="GIFs">
        {offer.onSignIn
            ? <div className="gif-picker-locked">
                <span className="gif-picker-lock"><LockIcon size={18}/></span>
                <button type="button" className="button button-mini button-action" onClick={offer.onSignIn}>
                    Sign in to send GIFs
                </button>
                <Close onClose={onClose}/>
            </div>
            : <Search offer={offer} onPick={onPick} onClose={onClose}/>}
    </div>
}

function Search({offer, onPick, onClose}: GifPickerProps) {
    const [typed, setTyped] = useState("")
    const found = useGifSearch(offer.search, typed, offer.locale, offer.customer)
    const end = useRef<HTMLDivElement>(null)

    const {loadMore} = found
    useEffect(() => {
        const sentinel = end.current
        if (!sentinel || typeof IntersectionObserver === "undefined") return

        const observer = new IntersectionObserver(entries => {
            if (entries.some(entry => entry.isIntersecting)) loadMore()
        }, {root: sentinel.parentElement, rootMargin: "120px"})
        observer.observe(sentinel)
        return () => observer.disconnect()
    }, [loadMore])

    // Inside the message's form: Enter here must not send it.
    const stayOnEnter = (event: KeyboardEvent<HTMLInputElement>) => {
        if (event.key === "Enter") event.preventDefault()
    }

    return <>
        <div className="gif-picker-head">
            <input className="field gif-picker-search"
                   type="search"
                   value={typed}
                   autoComplete="off"
                   aria-label="Search KLIPY"
                   placeholder="Search KLIPY"
                   enterKeyHint="search"
                   onKeyDown={stayOnEnter}
                   onChange={e => setTyped(e.target.value)}/>
            <Close onClose={onClose}/>
        </div>

        <div className="gif-picker-grid">
            {found.choices.map(choice => <button type="button"
                                                 key={choice.slug}
                                                 className="gif-picker-choice"
                                                 aria-label={choice.title || "GIF"}
                                                 title={choice.title}
                                                 onClick={() => onPick(choice, found.query)}>
                <img src={choice.gif.url} alt="" loading="lazy" decoding="async"/>
            </button>)}
            <div ref={end} className="gif-picker-end" aria-hidden="true"/>
        </div>

        {found.status === 'loading' && <p className="gif-picker-note">Loading…</p>}
        {found.status === 'ready' && found.choices.length === 0 && <p className="gif-picker-note">No GIF found</p>}
        {found.status === 'failed' && <p className="gif-picker-note" role="alert">
            The GIFs could not be loaded.{" "}
            <button type="button" className="gif-picker-retry" onClick={found.retry}>Try again</button>
        </p>}
    </>
}

function Close({onClose}: {onClose: () => void}) {
    return <button type="button"
                   className="gif-picker-close"
                   aria-label="Close the GIFs"
                   title="Close the GIFs"
                   onClick={onClose}>
        <CloseIcon size={14}/>
    </button>
}
