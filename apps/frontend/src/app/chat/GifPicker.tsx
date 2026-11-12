import {KeyboardEvent, useEffect, useRef, useState} from "react";
import {GifChoice, GifQuery, GifSearch} from "../../backends/gifs.ts";
import {CloseIcon, LockIcon} from "../components/icons.tsx";
import {useEscape} from "../components/useDialog.ts";
import {useGifSearch} from "./useGifSearch.ts";
import {t} from "../../i18n/page.ts";
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

    return <div className="gif-picker" role="group" aria-label={t("chat.gifs")}>
        {offer.onSignIn
            ? <div className="gif-picker-locked">
                <span className="gif-picker-lock"><LockIcon size={18}/></span>
                <button type="button" className="button button-mini button-action" onClick={offer.onSignIn}>
                    {t("gif.signIn")}
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
                   aria-label={t("gif.search")}
                   placeholder={t("gif.search")}
                   enterKeyHint="search"
                   onKeyDown={stayOnEnter}
                   onChange={e => setTyped(e.target.value)}/>
            <Close onClose={onClose}/>
        </div>

        <div className="gif-picker-grid">
            {found.choices.map(choice => <button type="button"
                                                 key={choice.slug}
                                                 className="gif-picker-choice"
                                                 aria-label={choice.title || t("chat.gif")}
                                                 title={choice.title}
                                                 onClick={() => onPick(choice, found.query)}>
                <img src={choice.gif.url} alt="" loading="lazy" decoding="async"/>
            </button>)}
            <div ref={end} className="gif-picker-end" aria-hidden="true"/>
        </div>

        {found.status === 'loading' && <p className="gif-picker-note">{t("common.loading")}</p>}
        {found.status === 'ready' && found.choices.length === 0 && <p className="gif-picker-note">{t("gif.none")}</p>}
        {found.status === 'failed' && <p className="gif-picker-note" role="alert">
            {t("gif.failed")}{" "}
            <button type="button" className="gif-picker-retry" onClick={found.retry}>{t("common.retry")}</button>
        </p>}
    </>
}

function Close({onClose}: {onClose: () => void}) {
    return <button type="button"
                   className="gif-picker-close"
                   aria-label={t("gif.close")}
                   title={t("gif.close")}
                   onClick={onClose}>
        <CloseIcon size={14}/>
    </button>
}
