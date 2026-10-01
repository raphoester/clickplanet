import {useEffect, useRef} from 'react'
import {Countries} from '../../domain/countries.ts'
import CountryFlag from './CountryFlag.tsx'
import './NativeLandNote.css'

/** How long the line stays up. Matched to the `native-land-note` keyframes. */
export const NATIVE_LAND_NOTE_MS = 4000

export type NativeLandNoteProps = {
    /** The country whose own ground the click cleared. */
    ground: string

    /** A quiz is in the band at the top of the screen, and this gives it the room, as the bomb line does. */
    lowered?: boolean

    onDone: () => void
}

/**
 * One line when a click of this player's cleared a tile rather than taking it.
 * Native land takes two clicks, so a newcomer clicking a native tile abroad sees
 * it go blank instead of flip: this says that was the rule, not a click that
 * went wrong. Said the first few times only (see domain/clearNotes.ts); the
 * dust on the tile says it every time. Nothing here is clickable.
 */
export default function NativeLandNote({ground, lowered, onDone}: NativeLandNoteProps) {
    const name = Countries.get(ground)?.name ?? ground

    // Held in a ref so only a new clear restarts the countdown; see BombNews.
    const done = useRef(onDone)
    useEffect(() => {
        done.current = onDone
    }, [onDone])

    useEffect(() => {
        const timer = setTimeout(() => done.current(), NATIVE_LAND_NOTE_MS)
        return () => clearTimeout(timer)
    }, [ground])

    return <div className={`native-land-note${lowered ? " native-land-note--lowered" : ""}`} role="status" aria-live="polite">
        <div className="native-land-note-line">
            <CountryFlag code={ground}/>
            <span><strong>{name}</strong>'s native land takes two clicks. One more to take it.</span>
        </div>
    </div>
}
