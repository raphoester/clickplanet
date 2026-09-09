import {useCallback, useEffect, useState} from 'react';
import {CHAT_IDENTITY_STORAGE_KEY, ChatIdentity, resolveIdentity} from './chatIdentity.ts';

function readStoredIdentity(): string | null {
    try {
        return window.localStorage.getItem(CHAT_IDENTITY_STORAGE_KEY)
    } catch {
        return null
    }
}

export const useChatIdentity = () => {
    const [identity, setIdentity] = useState<ChatIdentity>(
        () => resolveIdentity(readStoredIdentity()),
    )

    useEffect(() => {
        try {
            window.localStorage.setItem(CHAT_IDENTITY_STORAGE_KEY, JSON.stringify(identity))
        } catch (e) {
            console.error("Could not persist the chat identity", e)
        }
    }, [identity])

    const setName = useCallback(
        (name: string) => setIdentity(current => ({...current, name: name.trim()})),
        [],
    )

    return {identity, setName}
}
