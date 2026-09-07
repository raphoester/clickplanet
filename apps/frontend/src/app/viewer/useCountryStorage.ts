import {useCallback, useEffect, useState} from 'react';
import {Country} from '../countries';
import {COUNTRY_STORAGE_KEY, resolveCountry} from '../countryPreference';
import {currentTimeZone} from './visitorCountry';

function readStoredCountry(): string | null {
    try {
        return window.localStorage.getItem(COUNTRY_STORAGE_KEY)
    } catch {
        return null // private mode, or storage disabled entirely
    }
}

export const useCountryStorage = () => {
    const [countryState, setCountry] = useState<Country>(
        () => resolveCountry(readStoredCountry(), currentTimeZone()),
    )

    useEffect(() => {
        try {
            window.localStorage.setItem(COUNTRY_STORAGE_KEY, JSON.stringify(countryState))
        } catch (e) {
            console.error("Could not persist the selected country", e)
        }
    }, [countryState])

    const handleSetCountry = useCallback((v: Country) => setCountry(v), [])

    return {countryState, handleSetCountry}
}
