import {useId, useState} from "react"
import {PlayerInfoBackend} from "../../backends/player.ts"
import {CountryTiles, PlayerTitle} from "../../backends/progression.ts"
import {Reaction} from "../../backends/chat.ts"
import {Country} from "../../domain/countries.ts"
import {takenCount} from "../../domain/standings.ts"
import Modal from "../components/Modal.tsx"
import CountryFlag from "../components/CountryFlag.tsx"
import {ChevronIcon} from "../components/icons.tsx"
import {authorStyle} from "../chat/authorStyle.ts"
import {REACTION_IMAGES} from "../chat/reactionsAsset.ts"
import {useWordFilter} from "../crazygames/wordFilter.ts"
import {days} from "../days.ts"
import {FrontList} from "../players/PlayerFronts.tsx"
import {usePlayerInfo} from "../players/usePlayerInfo.ts"
import {useOwnStreak} from "../badges/badgesStore.ts"
import TitleEmblem from "../titles/TitleEmblem.tsx"
import {ListenForClicks} from "../viewer/acceptedClicks.ts"
import {useOwnTakes} from "../viewer/useAcceptedClicks.ts"
import {youLabel} from "../youLabel.ts"
import type {YouPlaceProps} from "../Menu.tsx"
import {AccountState, AccountStore} from "./accountStore.ts"
import {useAccount} from "./useAccount.ts"
import {AccountSettings} from "./AccountPanel.tsx"
import ProfileAccount from "./ProfileAccount.tsx"
import ProfileTracks from "./ProfileTracks.tsx"
import {useFronts, useTitles} from "./useProgress.ts"
import DeleteAccountModal from "./DeleteAccountModal.tsx"
import "./ProfileModal.css"

type Ready = Extract<AccountState, {kind: "ready"}>

const FIRE = REACTION_IMAGES.get(Reaction.FIRE)?.url
const FRONTS_SHOWN = 3

const day = new Intl.DateTimeFormat(undefined, {dateStyle: "medium"})
const count = new Intl.NumberFormat()

export type ProfileModalProps = YouPlaceProps & {
    country?: Country
    onClose: () => void
}

// The desktop's "You". A signed-in player sees its own card at full width: the worn title and the stats on top,
// the medals, the tracks and the fronts under them, and the account beside. What is long (every rank of a track,
// every country fought) opens on demand, so the card reads at a glance. A guest sees only the sign-in.
export default function ProfileModal({account: store, linkedMultiplier, playerInfo, listenForClicks, clips, country, onClose}: ProfileModalProps) {
    const account = useAccount(store)
    const words = useWordFilter()
    const [confirmingDelete, setConfirmingDelete] = useState(false)

    if (account.kind !== "ready") return null

    // One dialog at a time: the confirmation takes the profile's place, as a report takes a player card's.
    if (confirmingDelete) {
        return <DeleteAccountModal linked={account.me.linked}
                                   busy={account.busy === "deleteAccount"}
                                   onConfirm={() => void store.deleteAccount().then(() => setConfirmingDelete(false))}
                                   onClose={() => setConfirmingDelete(false)}/>
    }

    if (account.me.linked.length === 0) {
        return <Modal title={youLabel(false)} className="profile-modal profile-modal--guest" onClose={onClose}>
            <AccountSettings state={account}
                             store={store}
                             linkedMultiplier={linkedMultiplier}
                             clips={clips}
                             onDelete={() => setConfirmingDelete(true)}/>
        </Modal>
    }

    const name = account.username
    const title = name
        ? <span className="profile-modal-name" style={authorStyle({color: account.color, guest: false})}>{words.masked(name)}</span>
        : youLabel(true)

    return <Modal title={title} className="profile-modal" onClose={onClose}>
        <Profile account={account}
                 store={store}
                 country={country}
                 stats={name && account.accountId ? {backend: playerInfo, accountId: account.accountId} : undefined}
                 listenForClicks={listenForClicks}
                 aside={<ProfileAccount state={account}
                                        store={store}
                                        clips={clips}
                                        onDelete={() => setConfirmingDelete(true)}/>}/>
    </Modal>
}

type ProfileProps = {
    account: Ready
    store: AccountStore
    country?: Country
    stats?: {backend: PlayerInfoBackend, accountId: string}
    listenForClicks?: ListenForClicks
    aside: React.ReactNode
}

function Profile({account, store, country, stats, listenForClicks, aside}: ProfileProps) {
    const [titles, reread] = useTitles(store, account.me, listenForClicks)
    const fronts = useFronts(store, account.me, listenForClicks)
    const dashboard = titles.kind === "ready" ? titles.dashboard : undefined

    return <div className="profile">
        <div className="panel-box profile-hero">
            {dashboard?.worn && <span className="profile-hero-medal"><TitleEmblem title={dashboard.worn} size={112} ribbon/></span>}
            <div className="profile-hero-who">
                {dashboard?.worn && <WornTitle title={dashboard.worn}/>}
                {country && <span className="profile-hero-country">
                    <CountryFlag code={country.code}/>Plays for {country.name}
                </span>}
            </div>
            {stats && <OwnStats {...stats} listenForClicks={listenForClicks}/>}
        </div>

        <div className="profile-columns">
            <div className="profile-main">
                {titles.kind === "loading" && <p className="account-text" role="status">Loading…</p>}
                {titles.kind === "failed" && <p className="account-failure" role="alert">Your titles could not be loaded.</p>}
                {dashboard && <Medals worn={dashboard.worn} wearable={dashboard.wearable} store={store} onWorn={reread}/>}
                {dashboard && dashboard.tracks.length > 0 && <ProfileTracks tracks={dashboard.tracks}/>}
                {fronts && (fronts.playsFor.length > 0 || fronts.playsAgainst.length > 0) && <div className="profile-fronts">
                    <Front title="Fought for" countries={fronts.playsFor}/>
                    <Front title="Fought against" countries={fronts.playsAgainst}/>
                </div>}
            </div>
            {aside}
        </div>
    </div>
}

function WornTitle({title}: {title: PlayerTitle}) {
    return <>
        <span className="profile-hero-title">{title.name}</span>
        {title.rank && <span className="profile-hero-rank">
            {title.rank.trackName}, rank {title.rank.number} of {title.rank.count}
        </span>}
    </>
}

function OwnStats({backend, accountId, listenForClicks}: {backend: PlayerInfoBackend, accountId: string, listenForClicks?: ListenForClicks}) {
    const state = usePlayerInfo(backend, {accountId, guest: false})
    const takes = useOwnTakes(listenForClicks)
    useOwnStreak(accountId, state.kind === "ready" ? state.info.streakCurrent : undefined)
    if (state.kind !== "ready") return null

    const info = state.info
    return <dl className="profile-stats">
        <Stat label="Tiles taken" value={count.format(info.tilesTaken + takenCount(takes))}/>
        <Stat label="Streak" value={days(info.streakCurrent)} flame/>
        <Stat label="Best streak" value={days(info.streakBest)} flame faded/>
        {info.createdAt !== undefined && <Stat label="Playing since" value={day.format(info.createdAt)}/>}
    </dl>
}

function Stat({label, value, flame, faded}: {label: string, value: string, flame?: boolean, faded?: boolean}) {
    return <div className="profile-stat">
        <dt>{label}</dt>
        <dd>
            {flame && FIRE && <img src={FIRE}
                                   alt=""
                                   width={28}
                                   height={28}
                                   draggable={false}
                                   className={faded ? "profile-stat-flame profile-stat-flame--faded" : "profile-stat-flame"}/>}
            {value}
        </dd>
    </div>
}

function Medals({worn, wearable, store, onWorn}: {worn?: PlayerTitle, wearable: PlayerTitle[], store: AccountStore, onWorn: () => void}) {
    const [wearing, setWearing] = useState<string>()
    const titleId = useId()

    const wear = (id: string) => {
        setWearing(id)
        store.wearTitle(id)
            .then(onWorn, (e) => console.error("Could not wear the title", e))
            .finally(() => setWearing(undefined))
    }

    return <section className="profile-section" aria-labelledby={titleId}>
        <div className="profile-section-head">
            <h3 className="profile-section-title" id={titleId}>Titles</h3>
            {wearable.length > 0 && <span className="menu-label">Press a medal to wear it</span>}
        </div>
        {wearable.length === 0
            ? <p className="account-text">No title yet: the tracks below say how to earn one.</p>
            : <div className="profile-medals" role="radiogroup" aria-label="Wear a title">
                {wearable.map((title) => {
                    const checked = title.id === worn?.id
                    return <button key={title.id}
                                   type="button"
                                   role="radio"
                                   aria-checked={checked}
                                   aria-label={title.name}
                                   title={title.feat}
                                   className="panel-box profile-medal"
                                   disabled={wearing !== undefined}
                                   onClick={() => {
                                       if (!checked) wear(title.id)
                                   }}>
                        {checked && <span className="chip profile-medal-worn" aria-hidden="true">Worn</span>}
                        <TitleEmblem title={title} size={76} ribbon/>
                        <span className="profile-medal-name">{title.name}</span>
                    </button>
                })}
            </div>}
    </section>
}

function Front({title, countries}: {title: string, countries: CountryTiles[]}) {
    const [all, setAll] = useState(false)
    const headingId = useId()
    if (countries.length === 0) return null

    const shown = all ? countries : countries.slice(0, FRONTS_SHOWN)
    return <section className="panel-box profile-front" aria-labelledby={headingId}>
        <div className="profile-section-head">
            <h3 className="profile-front-title" id={headingId}>{title}</h3>
            <span className="menu-label">{countries.length === 1 ? "1 country" : `${countries.length} countries`}</span>
        </div>
        <FrontList countries={shown}/>
        {countries.length > FRONTS_SHOWN && <button type="button"
                                                    className="account-text-button profile-more"
                                                    aria-expanded={all}
                                                    onClick={() => setAll((was) => !was)}>
            {all ? "Show fewer" : `See all ${countries.length}`}<ChevronIcon size={14}/>
        </button>}
    </section>
}
