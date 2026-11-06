import {useEffect, useMemo, useRef, useState} from 'react';
import {
    AllegianceSource,
    BankFullError,
    Bomber,
    BonusListener,
    BonusLostError,
    ChargesRefresher,
    OwnershipsGetter,
    QuizMaster,
    Refiller,
    Shielder,
    TileClicker,
    TileHolders,
    LandmassReader,
    UpdatesListener,
} from "../../backends/backend.ts";
import News from "../news/News.tsx";
import ConnectionLost from "../components/ConnectionLost.tsx";
import {ConnectionSource} from "../../backends/connection.ts";
import {useConnection} from "../components/useConnection.ts";
import Quiz from "../quiz/Quiz.tsx";
import {useQuizMaster} from "../quiz/useQuizMaster.ts";
import {ChatRooms} from "../../backends/chat.ts";
import {GifSearch} from "../../backends/gifs.ts";
import ChatPanel from "../chat/ChatPanel.tsx";
import Menu, {BoardSheet, YouPlace} from "../Menu.tsx";
import {youLabel} from "../youLabel.ts";
import BonusAward from "../bonus/BonusAward.tsx";
import ClickBudgetMeter from "../components/ClickBudgetMeter.tsx";
import ClicksPanel from "../components/ClicksPanel.tsx";
import Inventory from "../bonus/Inventory.tsx";
import {useRefusedClicks} from "../clicks/refusedClicksStore.ts";
import SessionUnavailableModal from "../components/SessionUnavailableModal.tsx";
import VPNBlockedModal from "../components/VPNBlockedModal.tsx";
import BannedModal from "../components/BannedModal.tsx";
import WelcomeBackModal from "../account/WelcomeBackModal.tsx";
import SharePlace from "../share/SharePlace.tsx";
import SharePreview from "../share/SharePreview.tsx";
import {useSharePicture} from "../share/useSharePicture.ts";
import {shareStats} from "../../domain/shareCard.ts";
import {Countries, Country} from "../../domain/countries.ts";
import {Shot} from "../../domain/clip/camera.ts";
import {useAllegiance} from "../useAllegiance.ts";
import {ClickBudgetSource, now as budgetNow, tokensAt} from "../../backends/clickBudget.ts";
import {useClickBudget} from '../clicks/useClickBudget.ts';
import {useCountryStorage} from '../useCountryStorage.ts';
import {GlobeStatus, useGlobe} from '../viewer/useGlobe.ts';
import {useSound} from '../sound/useSound.ts';
import {useDisplaySettings} from '../settings/displaySettingsStore.ts';
import SettingsPlace from "../settings/SettingsPlace.tsx";
import AnthemControls from "../anthem/AnthemControls.tsx";
import {useAnthem} from "../anthem/useAnthem.ts";
import {AccountStore} from "../account/accountStore.ts";
import {useAccount} from "../account/useAccount.ts";
import {hueOf} from "../../domain/authorColor.ts";
import {PlayerInfoBackend, PresenceBackend} from "../../backends/player.ts";
import PlayerCardPlace from "../players/PlayerCardPlace.tsx";
import HolderCardPlace from "../players/HolderCardPlace.tsx";
import TerritoryCardPlace from "../territory/TerritoryCardPlace.tsx";
import {useTerritoryCard} from "../territory/useTerritoryCard.ts";
import {usePresence} from "../players/usePresence.ts";
import {useRosterFeed} from "../players/useRosterFeed.ts";
import {useTitlesEarned} from "../titles/useTitlesEarned.ts";
import {TitlesListener} from "../../backends/progression.ts";
import {useBadges} from "../badges/badgesStore.ts";
import {useBadgesFeed} from "../badges/useBadgesFeed.ts";
import {BadgesBackend} from "../../backends/badges.ts";
import {NameplatesBackend} from "../../backends/nameplates.ts";
import {useNameplatesFeed} from "../nameplates/useNameplatesFeed.ts";
import {useOwnNameplate} from "../nameplates/nameplatesStore.ts";
import {onlineCount, useRoster} from "../players/rosterStore.ts";
import SignInPitchModal from "../account/SignInPitchModal.tsx";
import SignInNudgeModal from "../account/SignInNudgeModal.tsx";
import {useSignInNudge} from "../account/useSignInNudge.ts";
import {useSignInRefill} from "../account/useSignInRefill.ts";
import {LeaderboardEntry, rankOf} from "../../domain/leaderboard.ts";
import {HallOfFameSeasons} from "../season/HallOfFame.tsx";
import {useCompact} from "../compact.ts";
import Sheet from "../hud/Sheet.tsx";
import StatusBar from "../hud/StatusBar.tsx";
import TabBar from "../hud/TabBar.tsx";
import {useScreen} from "../hud/screenStore.ts";
import MissionsPlace, {MissionsPlaceProps} from "../missions/MissionsPlace.tsx";
import {useMissions} from "../missions/useMissions.ts";
import {Mission, MissionsBackend} from "../../backends/missions.ts";
import {unseen} from "../../domain/missions.ts";
import MissionsDone from "../missions/MissionsDone.tsx";
import {useSeenMissions} from "../missions/seenMissionsStore.ts";
import {usePaidMissions} from "../missions/usePaidMissions.ts"
import {useMissionTargets} from "../missions/useMissionTargets.ts"
import {useMissionProgress} from "../missions/useMissionProgress.ts"
import {boardsOf, StandingsBackend} from "../../backends/standings.ts";
import {SupportBackend} from "../../backends/support.ts";
import {pageContextOf} from "../support/pageContext.ts";
import {BoardStandings, BoardView} from "../standings/BoardViews.tsx";
import {useRaceFeed} from "../standings/useRaceFeed.ts";
import {useRace} from "../standings/raceStore.ts";
import {TurnBackend} from "../../backends/turn.ts";
import {Battle, battleKeyOf} from "../../domain/battle.ts";
import {useAnnouncements} from "../announcements/announcementsStore.ts";
import Moments from "../announcements/Moments.tsx";
import {runningSeason, useSchedule} from "../battle/scheduleStore.ts";
import {useScheduleFeed} from "../battle/useScheduleFeed.ts";
import ScheduleChip, {BattleSheet, SeasonSheet} from "../battle/ScheduleChip.tsx";
import {useMood} from "../battle/useMood.ts";
import {moodOf} from "../../domain/mood.ts";
import {useBattleOver} from "../battle/useBattleOver.ts";
import BattleOver from "../battle/BattleOver.tsx";
import {countryLines, CountryOrder, shownOrder} from "../../domain/race.ts";
import {endedBookOf} from "../../domain/debrief.ts";
import {acceptedClicks} from "../clicks/acceptedClicks.ts";
import {useGuidedTakes} from "../frontGuide.ts";
import {ClipsBackend, ClipsNameBackend} from "../../backends/clips.ts";
import {PromoBackend} from "../../backends/promo.ts";
import {useFlagMissionDone} from "../missions/useFlagMissionDone.ts";
import {CrazyGamesPage} from "../../backends/crazyGames.ts";
import {useCrazyGames, useGameplay, useInvites, useRoom} from "../crazygames/useCrazyGames.ts";
import InvitedModal from "../crazygames/InvitedModal.tsx";
import InviteFriends from "../share/InviteFriends.tsx";
import {DiscordAlerts} from "../alerts/discordAlerts.ts";
import {useDiscordAlerts} from "../alerts/useDiscordAlerts.ts";
import {useUnderAttack} from "../alerts/useUnderAttack.ts";
import {Halo} from "../viewer/atmosphere.ts";
import "../halloween/Halloween.css"
import "./PlayScreen.css"

const NO_LEADERBOARD: readonly LeaderboardEntry[] = []

export type Backends = {
    tileClicker: TileClicker
    ownershipsGetter: OwnershipsGetter
    updatesListener: UpdatesListener
    clickBudgetSource: ClickBudgetSource
    connection: ConnectionSource
    bonusListener: BonusListener
    quizMaster: QuizMaster
    bomber: Bomber
    refiller: Refiller
    shielder: Shielder
    allegiance: AllegianceSource
    chatBackend: ChatRooms
    account: AccountStore
    presence: PresenceBackend
    playerInfo: PlayerInfoBackend
    badges: BadgesBackend
    nameplates: NameplatesBackend
    titles: TitlesListener
    tileHolders: TileHolders
    landmasses: LandmassReader
    turn: TurnBackend
    standings: StandingsBackend
    support: SupportBackend
    missions: MissionsBackend
    chargesRefresher: ChargesRefresher
    clips: ClipsBackend & ClipsNameBackend
    // A battle's teaser cards.
    promo: PromoBackend
    crazyGames: CrazyGamesPage | undefined
    alerts: DiscordAlerts | undefined
    gifs: GifSearch | undefined
}

export type PlayScreenProps = {
    sharedCountry?: Country
    linkedShot?: Shot
    showFront?: boolean
    backends: Backends
}

export default function PlayScreen(props: PlayScreenProps) {
    const {backends} = props
    const container = useRef<HTMLDivElement>(null)
    const {countryState, handleSetCountry} = useCountryStorage(props.sharedCountry)
    const allegiance = useAllegiance(backends.allegiance)
    useEffect(() => {
        const held = allegiance && allegiance.countryId !== countryState.code ? Countries.get(allegiance.countryId) : undefined
        if (held) handleSetCountry(held)
    }, [allegiance, countryState.code, handleSetCountry])
    const invitation = useInvites(backends.crazyGames, allegiance, handleSetCountry)
    const {chatBackend} = backends
    const heldFlag = allegiance?.countryId
    const flagRoom = useMemo(() => {
        const country = heldFlag === undefined ? undefined : Countries.get(heldFlag)
        return country ? {country, backend: chatBackend.flagRoom(country.code)} : undefined
    }, [chatBackend, heldFlag])
    const invite = backends.crazyGames?.room.inviteLink
    const pickCountry = async (country: Country) => {
        if (country.code !== countryState.code) await backends.allegiance.pledgeAllegiance(country.code)
        handleSetCountry(country)
    }
    const clickBudget = useClickBudget(backends.clickBudgetSource, countryState.code)
    const connection = useConnection(backends.connection)
    const crazyGames = useCrazyGames(backends.crazyGames)
    const sound = useSound(crazyGames.silenced)
    const mapView = useDisplaySettings((settings) => settings.mapView)
    const rendering = useDisplaySettings((settings) => settings.rendering)
    const account = useAccount(backends.account)
    const username = account.kind === 'ready' ? account.username : undefined
    const ownAccountId = account.kind === 'ready' ? account.accountId : undefined
    const color = account.kind === 'ready' ? account.color : undefined
    const linked = account.kind === 'ready' && account.me.linked.length > 0
    const discordConnectable = account.kind === 'ready' && account.offered.includes("discord")
    const joiningDiscord = account.kind === 'ready' && account.joiningDiscord === true
    const discord = useDiscordAlerts(backends.alerts, ownAccountId, discordConnectable, joiningDiscord)

    usePresence(backends.presence, {countryCode: countryState.code, username, color})

    useQuizMaster(backends.quizMaster, countryState.code, sound.play)
    const announce = useAnnouncements((announcements) => announcements.announce)
    useBadgesFeed(backends.badges)
    useNameplatesFeed(backends.nameplates)
    useOwnNameplate(ownAccountId, username, color)
    useRosterFeed(backends.presence)
    useTitlesEarned(backends.titles, linked, (title) => {
        announce({kind: "title", title})
        // The title worn may follow the one earned: the next read of this player's badge says.
        if (ownAccountId) useBadges.getState().expire(ownAccountId)
    })
    const crowd = useRoster(onlineCount)
    useScheduleFeed(backends.turn, backends.standings)
    const turn = useSchedule((schedule) => schedule.turn)
    const battle = useSchedule((schedule) => schedule.battle)
    const seasons = useSchedule((schedule) => schedule.seasons)
    const debrief = useSchedule((schedule) => schedule.debrief)
    const season = useSchedule(runningSeason)
    const bookId = seasons.current?.bookId ?? endedBookOf(debrief)
    useRaceFeed(backends.standings, bookId, seasons.today?.roundId, battle?.roundId)
    const race = useRace((standings) => standings.race)
    const [accepted] = useState(acceptedClicks)
    useGuidedTakes(props.showFront ?? false, accepted.listenForClicks)
    const mood = moodOf(turn, clickBudget)
    const halloween = turn?.kind === "halloween"
    useMood(mood)
    const battleOver = useBattleOver(battle, debrief)
    const [countryOrder, setCountryOrder] = useState<CountryOrder>("points")
    const [pitchOpen, setPitchOpen] = useState(false)
    const guest = account.kind === 'ready' && account.offered.length > 0 && account.me.linked.length === 0

    const compact = useCompact()
    const sheet = useScreen((screen) => screen.sheet)
    const openSheet = useScreen((screen) => screen.openSheet)
    const closeSheet = useScreen((screen) => screen.closeSheet)
    const closePopover = useScreen((screen) => screen.closePopover)
    const closeAll = useScreen((screen) => screen.closeAll)
    const [boardView, setBoardView] = useState<BoardView>("countries")

    useEffect(() => {
        closeAll()
    }, [compact, closeAll])

    const missionsState = useMissions(backends.missions, ownAccountId, allegiance?.countryId, accepted.listenForClicks)
    const guided = props.showFront ?? false
    const missionTargets = useMissionTargets(missionsState, guided)
    const missionsShown = !guided && missionsState.kind === "ready" ? missionsState.day : undefined
    const progress = useMissionProgress(missionsState, guided)

    const {
        status,
        tilesLoading,
        orbit,
        leaderboard,
        tileDeltas,
        tilesCount,
        capture,
        graphics,
        charges,
        rules,
        bombArmed,
        toggleBomb,
        switches,
        toggleSwitch,
        pointing,
        pointingTerritory,
        territoryOf,
        showTerritory,
        flyToLastLoss,
        flyToMission,
    } = useGlobe({
        container,
        linkedShot: props.linkedShot,
        showFront: props.showFront,
        tileClicker: backends.tileClicker,
        ownershipsGetter: backends.ownershipsGetter,
        updatesListener: backends.updatesListener,
        clickBudget: backends.clickBudgetSource,
        bonusListener: backends.bonusListener,
        bomber: backends.bomber,
        shielder: backends.shielder,
        playSound: sound.play,
        onClickAccepted: accepted.record,
        country: countryState,
        crowd,
        clickHue: hueOf(color),
        mapView,
        halo: haloOf(battle, halloween),
        bonusLook: halloween ? "pumpkin" : "box",
        missions: missionTargets,
        rendering,
    })

    useGameplay(backends.crazyGames, status.state === 'ready')
    useRoom(backends.crazyGames, status.state === 'ready', countryState.code)

    const territoryCard = useTerritoryCard(pointingTerritory, territoryOf, showTerritory)

    useUnderAttack(countryState.code, tileDeltas, accepted.listenForClicks)
    const refusals = useRefusedClicks((refused) => refused.refusals)

    const spendRefill = () => {
        if (clickBudget && tokensAt(clickBudget, budgetNow()) >= clickBudget.capacity) return false
        backends.refiller.useRefill(countryState.code).catch((e) => {
            if (e instanceof BankFullError || e instanceof BonusLostError) return
            console.error("could not use the refill", e)
        })
        return true
    }

    const signInRefill = rules?.signInRefill === true
    // CrazyGames forbids a sign-in that pops up or is the main call to action.
    const nudge = useSignInNudge(
        !backends.crazyGames && guest && clickBudget?.linkedMultiplier !== undefined && !clickBudget.frozen && !charges.refill && !pitchOpen,
        refusals, clickBudget?.fastStart === true)
    useSignInRefill(account.kind === 'ready' && account.guestLinked === true && signInRefill, charges.refill,
        () => void backends.chargesRefresher.refreshCharges().catch((e) => console.error("could not read the charges held", e)),
        () => void spendRefill())

    const order = shownOrder(race, countryOrder)
    const lines = countryLines(leaderboard, race, order)
    const anthem = useAnthem(status.state === 'ready' ? lines : NO_LEADERBOARD, order, crazyGames.silenced)

    const stats = shareStats(lines, countryState)
    const {shot, taking, take, discard} = useSharePicture(capture, stats)

    const openPitch = guest ? () => {
        closeSheet()
        closePopover("clicks")
        setPitchOpen(true)
    } : undefined

    const toll = rules?.toll ?? []
    const held = leaderboard.find((entry) => entry.country.code === countryState.code)?.tiles ?? 0
    const boards = useMemo(() => bookId ? boardsOf(backends.standings, bookId) : undefined, [backends.standings, bookId])
    const standings: BoardStandings | undefined = boards && {
        backend: boards,
        caller: {accountId: ownAccountId, username, linked},
        listenForClicks: accepted.listenForClicks,
        onSignIn: clickBudget?.linkedMultiplier ? openPitch : undefined,
        view: boardView,
        onView: setBoardView,
    }
    const teased = season ?? seasons.next
    const hallOfFame: HallOfFameSeasons | undefined = teased || seasons.ended.length > 0
        ? {teased, ended: seasons.ended, standings: backends.standings, turn: backends.turn}
        : undefined

    const board = {
        country: countryState,
        setCountry: pickCountry,
        allegiance,
        leaderboard,
        tileDeltas,
        tilesCount,
        anthem: <AnthemControls anthem={anthem}/>,
        standings,
        countryOrder,
        onCountryOrder: setCountryOrder,
        hallOfFame,
    }
    const [missionsDone, setMissionsDone] = useState<Mission[]>()
    usePaidMissions(missionsState, setMissionsDone)
    useFlagMissionDone(missionsState, (mission) => {
        if (allegiance) announce({kind: "flagMission", mission, country: allegiance.countryId})
    })
    const missions: MissionsPlaceProps = {
        state: missionsState,
        onSignIn: guest ? openPitch ?? (() => openSheet("you")) : undefined,
        onGo: (target) => {
            closeSheet()
            flyToMission(target)
        },
    }
    const missionsNews = useSeenMissions((seenMissions) => missionsState.kind === "ready" ? unseen(missionsState.day.missions, seenMissions.seen) : 0)

    const you = {
        account: backends.account,
        linkedMultiplier: clickBudget?.linkedMultiplier,
        playerInfo: backends.playerInfo,
        listenForClicks: accepted.listenForClicks,
        clips: backends.clips,
    }
    const share = {
        onTakePicture: () => {
            closeSheet()
            take()
        },
        taking,
    }
    const settings = {
        sound: {preview: sound.preview},
        support: backends.support,
        pageContext: () => pageContextOf(window, graphics(), connection === "down"),
        discord,
    }
    const clicks = <ClicksPanel budget={clickBudget}
                                countryName={countryState.name}
                                share={tilesCount > 0 ? held / tilesCount : undefined}
                                toll={toll}
                                onSignIn={openPitch}/>

    const ready = status.state === 'ready'

    return <>
        <div ref={container} className="viewer-canvas"/>

        {!ready && <StatusCard status={status}/>}
        {ready && !orbit && tilesLoading !== undefined && <StatusCard status={{state: 'loading', territories: tilesLoading}}/>}

        {ready && !compact && <Menu {...board} {...you} {...share} {...settings}
                                    missions={missions}
                                    missionsNews={missionsNews}
                                    invite={invite}/>}

        {ready && !compact && <ScheduleChip compact={false}/>}

        {ready && compact && <StatusBar country={countryState}
                                        rank={rankOf(lines, countryState)}
                                        trailing={<ScheduleChip compact/>}/>}

        {ready && <ClickBudgetMeter budget={clickBudget}
                                    frozen={mood === "frozen" ? {nextSeason: seasons.next, country: countryState} : undefined}
                                    invite={invite}
                                    discord={discord}
                                    onSignIn={openPitch}
                                    compact={compact}
                                    popover={clicks}>
            <Inventory charges={charges}
                       rules={rules}
                       switches={switches}
                       onToggle={toggleSwitch}
                       bombArmed={bombArmed}
                       onToggleBomb={toggleBomb}
                       onUseRefill={spendRefill}
                       onToggleShield={() => toggleSwitch("shield")}/>
        </ClickBudgetMeter>}

        {ready && crazyGames.chat && <ChatPanel
            backend={backends.chatBackend}
            flagRoom={flagRoom}
            onSignIn={clickBudget?.linkedMultiplier ? openPitch : undefined}
            onLeftRoom={() => void backends.allegiance.refreshAllegiance().catch((e) => console.error("could not read the flag played for", e))}
            country={countryState}
            playSound={sound.play}
            accountId={ownAccountId}
            username={username}
            reporter={backends.support}
            compact={compact}
            open={compact ? sheet === "chat" : undefined}
            onOpenChange={compact ? (open) => open ? openSheet("chat") : closeSheet() : undefined}
            discord={discord}
            gifs={backends.gifs}
        />}

        {ready && compact && <>
            {sheet === "board" && <BoardSheet {...board} onClose={closeSheet}/>}
            {sheet === "missions" && <Sheet title="Missions" onClose={closeSheet}>
                <MissionsPlace {...missions}/>
            </Sheet>}
            {sheet === "you" && account.kind === 'ready' && <Sheet title={youLabel(linked)} onClose={closeSheet}>
                <YouPlace {...you}/>
            </Sheet>}
            {sheet === "share" && <Sheet title="Share" onClose={closeSheet}>
                <SharePlace stats={stats} {...share}/>
            </Sheet>}
            {sheet === "invite" && invite && <Sheet title="Invite" onClose={closeSheet}>
                <InviteFriends country={countryState} invite={invite}/>
            </Sheet>}
            {sheet === "settings" && <Sheet title="Settings" onClose={closeSheet}>
                <SettingsPlace {...settings}/>
            </Sheet>}
            {sheet === "season" && <SeasonSheet onClose={closeSheet}/>}
            {sheet === "battle" && <BattleSheet onClose={closeSheet}/>}
            {sheet === "clicks" && <Sheet title="Your clicks" onClose={closeSheet}>{clicks}</Sheet>}

            <TabBar chat={crazyGames.chat}
                    invite={invite !== undefined}
                    you={account.kind === 'ready' ? (linked ? "player" : "guest") : undefined}
                    missions={missionsNews}/>
        </>}

        {shot && <SharePreview shot={shot}
                               stats={stats}
                               onClose={discard}/>}

        {pitchOpen && guest && clickBudget?.linkedMultiplier && <SignInPitchModal
            state={account}
            store={backends.account}
            multiplier={clickBudget.linkedMultiplier}
            onClose={() => setPitchOpen(false)}/>}

        {nudge.open && guest && clickBudget?.linkedMultiplier && <SignInNudgeModal
            state={account}
            store={backends.account}
            nudge={nudge.open}
            multiplier={clickBudget.linkedMultiplier}
            fastStart={clickBudget.fastStart === true}
            fullBank={signInRefill}
            onClose={nudge.close}/>}

        {invitation.asked && allegiance && <InvitedModal key={invitation.asked.code}
                                                         friend={invitation.asked}
                                                         held={allegiance}
                                                         onSwitch={() => pickCountry(invitation.asked!)}
                                                         onClose={invitation.answer}/>}

        <HolderCardPlace holders={backends.tileHolders}
                         players={backends.playerInfo}
                         pointing={pointing}
                         accountId={ownAccountId}
                         missions={missionsShown}/>

        <TerritoryCardPlace shown={territoryCard.shown}
                            flag={countryState.code}
                            reader={backends.landmasses}
                            players={backends.playerInfo}
                            missions={missionsShown}
                            onHold={territoryCard.hold}
                            onClose={territoryCard.dismiss}/>

        <PlayerCardPlace backend={backends.playerInfo}
                         accountId={ownAccountId}
                         reporter={backends.support}/>

        <Moments shown={ready}
                 guided={props.showFront ?? false}
                 promo={backends.promo}
                 listenForClicks={accepted.listenForClicks}
                 clips={backends.clips}
                 clipsAccount={linked && username !== undefined ? ownAccountId : undefined}
                 countryCode={countryState.code}
                 accountId={ownAccountId}
                 play={sound.play}
                 onWear={backends.account.wearTitle}/>

        {missionsDone && rules && <MissionsDone missions={missionsDone}
                                                rules={rules}
                                                onDone={() => setMissionsDone(undefined)}/>}

        <BonusAward/>

        {ready && connection === "down" && <ConnectionLost/>}

        <News flag={countryState.code}
              connect={discord.connect}
              goThere={flyToLastLoss}
              progress={missionsDone ? undefined : progress}/>

        {battleOver.over && <BattleOver key={battleKeyOf(battleOver.over)}
                                        awaiting={battleOver.awaiting}
                                        onDone={battleOver.dismiss}/>}

        <Quiz/>

        <VPNBlockedModal/>
        <BannedModal/>
        {account.kind === 'ready' && account.restored &&
            <WelcomeBackModal onClose={() => backends.account.welcomedBack()}/>}

        <SessionUnavailableModal/>
    </>
}

function haloOf(battle: Battle | undefined, halloween: boolean): Halo {
    if (battle) return "war"
    if (halloween) return "haunt"
    return "calm"
}

function StatusCard({status}: {status: GlobeStatus}) {
    return <div className="viewer-status">
        <div className="viewer-status-card panel" role="status">
            {status.state === 'loading'
                ? <LoadingCard territories={status.territories}/>
                : status.state === 'failed' && <>
                    <h3>The globe could not be loaded</h3>
                    <p>{status.message}</p>
                </>}
        </div>
    </div>
}

function LoadingCard({territories}: {territories: number | undefined}) {
    const percent = Math.round((territories ?? 0) * 100)
    return <>
        <div className="viewer-status-spinner"/>
        <h3>{territories === undefined ? "Loading the planet…" : "Loading territories…"}</h3>
        <div className="viewer-status-progress"
             role="progressbar"
             aria-label="Territories loaded"
             aria-valuemin={0}
             aria-valuemax={100}
             aria-valuenow={percent}>
            <div className="viewer-status-progress-fill" style={{width: `${percent}%`}}/>
        </div>
    </>
}
