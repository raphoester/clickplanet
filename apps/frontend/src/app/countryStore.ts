import {useEffect} from 'react';
import {create} from 'zustand';
import {Country} from '../domain/countries';
import {COUNTRY_STORAGE_KEY, resolveCountry} from './countryPreference';
import {currentTimeZone} from './visitorCountry';

type CountryStore = {
    country: Country
    choose: (country: Country) => void
}

function readStoredCountry(): string | null {
    try {
        return window.localStorage.getItem(COUNTRY_STORAGE_KEY)
    } catch {
        return null
    }
}

export const useCountry = create<CountryStore>()((set) => ({
    country: resolveCountry(readStoredCountry(), currentTimeZone()),
    choose: (country) => set({country}),
}))

// Kept from the page's first render, not on import: the home page and the front guide tell a first visit by its absence.
export function useKeepCountry(): void {
    const country = useCountry((store) => store.country)

    useEffect(() => {
        try {
            window.localStorage.setItem(COUNTRY_STORAGE_KEY, JSON.stringify(country))
        } catch (e) {
            console.error("Could not persist the selected country", e)
        }
    }, [country])
}
