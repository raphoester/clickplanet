import {useEffect, useState} from 'react';
import {DISPLAY_SETTINGS_STORAGE_KEY, DisplaySettings, parseDisplaySettings} from '../../domain/displaySettings.ts';

function readStoredSettings(): string | null {
    try {
        return window.localStorage.getItem(DISPLAY_SETTINGS_STORAGE_KEY)
    } catch {
        return null
    }
}

export function useDisplaySettings() {
    const [settings, setSettings] = useState<DisplaySettings>(() => parseDisplaySettings(readStoredSettings()))

    useEffect(() => {
        try {
            window.localStorage.setItem(DISPLAY_SETTINGS_STORAGE_KEY, JSON.stringify(settings))
        } catch (e) {
            console.error("Could not persist the display settings", e)
        }
    }, [settings])

    return {settings, setSettings}
}
