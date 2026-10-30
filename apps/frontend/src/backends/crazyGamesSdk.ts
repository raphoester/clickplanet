import {CrazyGamesPage, CrazyGamesSettings, DEFAULT_CRAZYGAMES_SETTINGS} from "./crazyGames.ts"

const SDK_URL = "https://sdk.crazygames.com/crazygames-sdk-v3.js"

type SdkSettings = {disableChat: boolean, muteAudio: boolean}

type SdkError = {code?: string}

export type Sdk = {
    init(): Promise<void>
    environment: "local" | "crazygames" | "disabled"
    user: {
        isUserAccountAvailable: boolean
        getUserToken(options: {signingKeyType: "IN_GAME_AUTH"}): Promise<string>
        showAuthPrompt(): Promise<unknown>
        addAuthListener(listener: (user: unknown) => void): void
        removeAuthListener(listener: (user: unknown) => void): void
    }
    game: {
        settings: SdkSettings
        addSettingsChangeListener(listener: (settings: SdkSettings) => void): void
        removeSettingsChangeListener(listener: (settings: SdkSettings) => void): void
        loadingStart(): void
        loadingStop(): void
        gameplayStart(): void
    }
}

declare global {
    interface Window {
        CrazyGames?: {SDK: Sdk}
    }
}

function loadScript(): Promise<Sdk> {
    return new Promise((resolve, reject) => {
        const script = document.createElement("script")
        script.src = SDK_URL
        script.onload = () => window.CrazyGames ? resolve(window.CrazyGames.SDK) : reject(new Error("the CrazyGames SDK loaded without its API"))
        script.onerror = () => reject(new Error("the CrazyGames SDK could not be loaded"))
        document.head.appendChild(script)
    })
}

function settingsOf(sdk: SdkSettings): CrazyGamesSettings {
    return {chat: !sdk.disableChat, audio: !sdk.muteAudio}
}

// The game never waits on the SDK: a call made before it is ready runs once it is, and one that cannot is a guest's.
export function connectCrazyGames(load: () => Promise<Sdk> = loadScript): CrazyGamesPage {
    const ready: Promise<Sdk | undefined> = load()
        .then(async (sdk) => {
            await sdk.init()
            if (sdk.environment === "disabled") return undefined
            sdk.game.loadingStart()
            return sdk
        })
        .catch((e) => {
            console.error("The CrazyGames SDK is not there: the page plays as a guest", e)
            return undefined
        })
    let current: Sdk | undefined
    void ready.then((sdk) => current = sdk)

    return {
        user: {
            token: async () => {
                const sdk = await ready
                if (!sdk?.user.isUserAccountAvailable) return undefined
                try {
                    return await sdk.user.getUserToken({signingKeyType: "IN_GAME_AUTH"})
                } catch (e) {
                    if ((e as SdkError)?.code !== "userNotAuthenticated") console.error("CrazyGames gave no token", e)
                    return undefined
                }
            },
            prompt: async () => {
                const sdk = await ready
                if (!sdk?.user.isUserAccountAvailable) return false
                try {
                    return await sdk.user.showAuthPrompt() != null
                } catch (e) {
                    return (e as SdkError)?.code === "userAlreadySignedIn"
                }
            },
            listenForSignIn: (listener) => {
                const heard = (user: unknown) => {
                    if (user) listener()
                }
                let off = false
                void ready.then((sdk) => {
                    if (!off) sdk?.user.addAuthListener(heard)
                })
                return () => {
                    off = true
                    current?.user.removeAuthListener(heard)
                }
            },
        },
        settings: () => current ? settingsOf(current.game.settings) : DEFAULT_CRAZYGAMES_SETTINGS,
        listenForSettings: (listener) => {
            const heard = (settings: SdkSettings) => listener(settingsOf(settings))
            let off = false
            void ready.then((sdk) => {
                if (off || !sdk) return
                heard(sdk.game.settings)
                sdk.game.addSettingsChangeListener(heard)
            })
            return () => {
                off = true
                current?.game.removeSettingsChangeListener(heard)
            }
        },
        started: () => void ready.then((sdk) => {
            sdk?.game.loadingStop()
            sdk?.game.gameplayStart()
        }),
    }
}
