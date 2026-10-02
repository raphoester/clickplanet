import {useEffect, useRef} from 'react'
import {Countries} from '../../domain/countries.ts'
import CountryFlag from './CountryFlag.tsx'
import './NativeLandNote.css'

export const NATIVE_LAND_NOTE_MS = 4000

export type NativeLandNoteProps = {
    ground: string

    lowered?: boolean

    onDone: () => void
}

export default function NativeLandNote({ground, lowered, onDone}: NativeLandNoteProps) {
    const name = Countries.get(ground)?.name ?? ground

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
