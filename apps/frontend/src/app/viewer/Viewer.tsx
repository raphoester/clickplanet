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
import BombNews from "../components/BombNews.tsx";
import FortifyNews from "../components/FortifyNews.tsx";
import FortifyCard from "../components/FortifyCard.tsx";
import FortifyHint from "../components/FortifyHint.tsx";
import {useFortifyGuide} from "../components/useFortifyGuide.ts";
import {Fortified} from "../../domain/fortify.ts";
import ConnectionLost from "../components/ConnectionLost.tsx";
import {ConnectionSource} from "../../backends/connection.ts";
import {useConnection} from "./useConnection.ts";
import Quiz from "../quiz/Quiz.tsx";
import {useQuizMaster} from "../quiz/useQuizMaster.ts";
import {ChatRooms} from "../../backends/chat.ts";
import {GifSearch} from "../../backends/gifs.ts";
import ChatPanel from "../chat/ChatPanel.tsx";
import Menu, {BoardSheet, YouPlace} from "../Menu.tsx";
import {youLabel} from "../youLabel.ts";
import BonusAward from "../components/BonusAward.tsx";
import ClickBudgetMeter from "../components/ClickBudgetMeter.tsx";
import ClicksPanel from "../components/ClicksPanel.tsx";
import Inventory from "../components/Inventory.tsx";
import {useBonusGuide} from "../components/bonusGuideStore.ts";
import {isFirstWin} from "../../domain/bonusGuide.ts";
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
import {useAllegiance} from "./useAllegiance.ts";
import {heldLine} from "../../domain/allegiance.ts";
import {ClickBudgetSource, now as budgetNow, tokensAt} from "../../backends/clickBudget.ts";
import {useClickBudget} from './useClickBudget.ts';
import {useCountryStorage} from './useCountryStorage.ts';
import {GlobeStatus, useGlobe} from './useGlobe.ts';
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
import HolderCard from "../players/HolderCard.tsx";
import TerritoryCard from "../territory/TerritoryCard.tsx";
import {useTerritoryCard} from "../territory/useTerritoryCard.ts";
import {useTerritoryPlayers} from "../territory/useTerritoryPlayers.ts";
import {useTileHolder} from "../players/useTileHolder.ts";
import {usePresence} from "../players/usePresence.ts";
import {useRoster} from "../players/useRoster.ts";
import SignInPitchModal from "../account/SignInPitchModal.tsx";
import SignInNudgeModal from "../account/SignInNudgeModal.tsx";
import {useSignInNudge} from "../account/useSignInNudge.ts";
import {useSignInRefill} from "../account/useSignInRefill.ts";
import {LeaderboardEntry, rankOf} from "../../domain/leaderboard.ts";
import {useSeason} from "../season/useSeason.ts";
import SeasonChip, {ChampionChip, SeasonDetails} from "../season/SeasonChip.tsx";
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
import {usePaidMissions} from "../missions/usePaidMissions.ts";
import {boardsOf, StandingsBackend} from "../../backends/standings.ts";
import {SupportBackend} from "../../backends/support.ts";
import {pageContextOf} from "../support/pageContext.ts";
import {BoardStandings, BoardView} from "../standings/BoardViews.tsx";
import {useRace} from "../standings/useRace.ts";
import {useDebriefed} from "../season/useDebriefed.ts";
import {TurnBackend} from "../../backends/turn.ts";
import {Battle, battleKeyOf, battleOf, finaleOf, nextBattleOf} from "../../domain/battle.ts";
import {useAnnouncements} from "../announcements/announcementsStore.ts";
import Moments from "../announcements/Moments.tsx";
import {useSchedule} from "../battle/useSchedule.ts";
import {useMood} from "../battle/useMood.ts";
import {moodOf} from "../../domain/mood.ts";
import {useBattleOver} from "../battle/useBattleOver.ts";
import BattleChip, {BattleDetails} from "../battle/BattleChip.tsx";
import BattleOver from "../battle/BattleOver.tsx";
import {reigningChampionOf} from "../../domain/champion.ts";
import {countryLines, CountryOrder, shownOrder} from "../../domain/race.ts";
import {endedBookOf} from "../../domain/debrief.ts";
import {acceptedClicks} from "./acceptedClicks.ts";
import {useGuidedTakes} from "./frontGuide.ts";
import {ClipsBackend, ClipsNameBackend} from "../../backends/clips.ts";
import {useFlagMissionDone} from "../missions/useFlagMissionDone.ts";
import {CrazyGamesPage} from "../../backends/crazyGames.ts";
import {useCrazyGames, useGameplay, useInvites, useRoom} from "../crazygames/useCrazyGames.ts";
import InvitedModal from "../crazygames/InvitedModal.tsx";
import InviteFriends from "../share/InviteFriends.tsx";
import {DiscordAlerts} from "../alerts/discordAlerts.ts";
import {useDiscordAlerts} from "../alerts/useDiscordAlerts.ts";
import {useUnderAttack} from "../alerts/useUnderAttack.ts";
import UnderAttack from "../alerts/UnderAttack.tsx";
import {Halo} from "./atmosphere.ts";
import "../halloween/Halloween.css"
import "./Viewer.css"

const NO_LEADERBOARD: readonly LeaderboardEntry[] = []

// After the wave has run over the land.
const FORTIFY_CARD_DELAY_MS = 1600

export type ViewerProps = {
    sharedCountry?: Country
    linkedShot?: Shot
    showFront?: boolean
    tileClicker: TileClicker
    ownershipsGetter: OwnershipsGetter
    updatesListener: UpdatesListener
    clickBudgetSource?: ClickBudgetSource
    connection?: ConnectionSource
    bonusListener?: BonusListener
    quizMaster?: QuizMaster
    bomber?: Bomber
    refiller?: Refiller
    shielder?: Shielder
    allegiance?: AllegianceSource
    chatBackend?: ChatRooms
    account?: AccountStore
    presence?: PresenceBackend
    playerInfo?: PlayerInfoBackend
    tileHolders?: TileHolders
    landmasses?: LandmassReader
    turn?: TurnBackend
    standings?: StandingsBackend
    support?: SupportBackend
    missions?: MissionsBackend
    chargesRefresher?: ChargesRefresher
    clips?: ClipsBackend & ClipsNameBackend
    crazyGames?: CrazyGamesPage
    alerts?: DiscordAlerts
    gifs?: GifSearch
}

export default function Viewer(props: ViewerProps) {
    const container = useRef<HTMLDivElement>(null)
    const {countryState, handleSetCountry} = useCountryStorage(props.sharedCountry)
    const allegiance = useAllegiance(props.allegiance)
    useEffect(() => {
        const held = allegiance && allegiance.countryId !== countryState.code ? Countries.get(allegiance.countryId) : undefined
        if (held) handleSetCountry(held)
    }, [allegiance, countryState.code, handleSetCountry])
    const invitation = useInvites(props.crazyGames, allegiance, handleSetCountry)
    const {chatBackend} = props
    const heldFlag = allegiance?.countryId
    const flagRoom = useMemo(() => {
        const country = heldFlag === undefined ? undefined : Countries.get(heldFlag)
        return country && chatBackend ? {country, backend: chatBackend.flagRoom(country.code)} : undefined
    }, [chatBackend, heldFlag])
    const invite = props.crazyGames?.room.inviteLink
    const pickCountry = async (country: Country) => {
        if (country.code !== countryState.code) await props.allegiance?.pledgeAllegiance(country.code)
        handleSetCountry(country)
    }
    const clickBudget = useClickBudget(props.clickBudgetSource, countryState.code)
    const connection = useConnection(props.connection)
    const crazyGames = useCrazyGames(props.crazyGames)
    const sound = useSound(crazyGames.silenced)
    const mapView = useDisplaySettings((settings) => settings.mapView)
    const rendering = useDisplaySettings((settings) => settings.rendering)
    const account = useAccount(props.account)
    const username = account.kind === 'ready' ? account.username : undefined
    const ownAccountId = account.kind === 'ready' ? account.accountId : undefined
    const color = account.kind === 'ready' ? account.color : undefined
    const linked = account.kind === 'ready' && account.me.linked.length > 0
    const discordConnectable = account.kind === 'ready' && account.offered.includes("discord")
    const discord = useDiscordAlerts(props.alerts, ownAccountId, discordConnectable)

    usePresence(props.presence, {countryCode: countryState.code, username, color})

    useQuizMaster(props.quizMaster, countryState.code, sound.play)
    const announce = useAnnouncements((announcements) => announcements.announce)
    const roster = useRoster(props.presence, (title) => announce({kind: "title", title}))
    const schedule = useSchedule(props.turn)
    const season = useSeason(schedule.seasons.current)
    const bookId = schedule.seasons.current?.bookId ?? endedBookOf(schedule.debrief)
    const dayEndsAt = schedule.seasons.today?.endsAt
    const battle = battleOf(schedule.turn)
    const race = useRace(props.standings, bookId, schedule.seasons.today?.roundId, battle?.roundId)
    const closed = useDebriefed(props.standings, schedule.debrief)
    const nextBattle = nextBattleOf(schedule.battles, Date.now())
    const finale = finaleOf(schedule.battles, schedule.turn, Date.now())
    const [accepted] = useState(acceptedClicks)
    useGuidedTakes(props.showFront ?? false, accepted.listenForClicks)
    const mood = moodOf(schedule.turn, clickBudget)
    const halloween = schedule.turn?.kind === "halloween"
    useMood(mood)
    const battleOver = useBattleOver(battle, schedule.debrief)
    const champion = reigningChampionOf(season, closed)
    const [countryOrder, setCountryOrder] = useState<CountryOrder>("season")
    const [pitchOpen, setPitchOpen] = useState(false)
    const guest = account.kind === 'ready' && account.offered.length > 0 && account.me.linked.length === 0

    const compact = useCompact()
    const sheet = useScreen((screen) => screen.sheet)
    const openSheet = useScreen((screen) => screen.openSheet)
    const closeSheet = useScreen((screen) => screen.closeSheet)
    const closePopover = useScreen((screen) => screen.closePopover)
    const closeAll = useScreen((screen) => screen.closeAll)
    const [unread, setUnread] = useState(0)
    const [boardView, setBoardView] = useState<BoardView>("countries")

    useEffect(() => {
        closeAll()
    }, [compact, closeAll])

    const {
        status,
        tilesLoading,
        orbit,
        leaderboard,
        tileDeltas,
        tilesCount,
        capture,
        graphics,
        refusals,
        vpnBlocked,
        dismissVPNBlocked,
        bannedUntil,
        dismissBanned,
        sessionUnavailable,
        dismissSessionUnavailable,
        lockedOut,
        award,
        dismissAward,
        charges,
        rules,
        bombArmed,
        toggleBomb,
        switches,
        toggleSwitch,
        notice,
        lastBomb,
        dismissBomb,
        lastFortified,
        dismissFortified,
        lastClose,
        dismissClose,
        pointing,
        pointingTerritory,
        territoryOf,
        showTerritory,
        flyToLastLoss,
    } = useGlobe({
        container,
        linkedShot: props.linkedShot,
        showFront: props.showFront,
        tileClicker: props.tileClicker,
        ownershipsGetter: props.ownershipsGetter,
        updatesListener: props.updatesListener,
        clickBudget: props.clickBudgetSource,
        bonusListener: props.bonusListener,
        bomber: props.bomber,
        shielder: props.shielder,
        playSound: sound.play,
        onClickAccepted: accepted.record,
        country: countryState,
        crowd: roster.kind === "ready" ? roster.entries.length : undefined,
        clickHue: hueOf(color),
        mapView,
        halo: haloOf(battle, halloween),
        bonusLook: halloween ? "pumpkin" : "box",
        rendering,
    })

    useGameplay(props.crazyGames, status.state === 'ready')
    useRoom(props.crazyGames, status.state === 'ready', countryState.code)

    const firstWin = useBonusGuide((bonus) => award !== undefined && isFirstWin(bonus.guide, award.kind))
    const markWon = useBonusGuide((bonus) => bonus.markWon)

    const tileHolder = useTileHolder(props.tileHolders, props.playerInfo, pointing, ownAccountId)
    const territoryCard = useTerritoryCard(pointingTerritory, territoryOf, showTerritory)
    const territoryPlayers = useTerritoryPlayers(props.landmasses, props.playerInfo, territoryCard.shown?.territory.landmass)

    const fortifyGuide = useFortifyGuide()
    const [fortifyCard, setFortifyCard] = useState<Fortified | undefined>()
    useEffect(() => {
        if (!lastFortified || fortifyGuide.seen) return
        const timer = setTimeout(() => setFortifyCard(lastFortified.fortified), FORTIFY_CARD_DELAY_MS)
        return () => clearTimeout(timer)
    }, [lastFortified, fortifyGuide.seen])
    const underAttack = useUnderAttack(countryState.code, tileDeltas, accepted.listenForClicks)
    // Moments show one at a time: the newest of them.
    const newest = [lastBomb, lastFortified, lastClose, underAttack.moment].reduce<{at: number} | undefined>(
        (latest, moment) => moment && (!latest || moment.at > latest.at) ? moment : latest, undefined)

    const refiller = props.refiller
    const spendRefill = refiller && (() => {
        if (clickBudget && tokensAt(clickBudget, budgetNow()) >= clickBudget.capacity) return false
        refiller.useRefill(countryState.code).catch((e) => {
            if (e instanceof BankFullError || e instanceof BonusLostError) return
            console.error("could not use the refill", e)
        })
        return true
    })

    const signInRefill = rules?.signInRefill === true
    // CrazyGames forbids a sign-in that pops up or is the main call to action.
    const nudge = useSignInNudge(
        !props.crazyGames && guest && clickBudget?.linkedMultiplier !== undefined && !clickBudget.frozen && !charges.refill && !pitchOpen,
        refusals)
    useSignInRefill(account.kind === 'ready' && account.guestLinked === true && signInRefill, charges.refill,
        () => void props.chargesRefresher?.refreshCharges().catch((e) => console.error("could not read the charges held", e)),
        () => void spendRefill?.())

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
    const boards = useMemo(() => props.standings && bookId ? boardsOf(props.standings, bookId) : undefined, [props.standings, bookId])
    const standings: BoardStandings | undefined = boards && {
        backend: boards,
        caller: {accountId: ownAccountId, username, color, linked},
        listenForClicks: accepted.listenForClicks,
        onSignIn: clickBudget?.linkedMultiplier ? openPitch : undefined,
        view: boardView,
        onView: setBoardView,
    }
    const teased = season ?? schedule.seasons.next
    const hallOfFame: HallOfFameSeasons | undefined = props.standings && props.turn && (teased || schedule.seasons.ended.length > 0)
        ? {teased, ended: schedule.seasons.ended, standings: props.standings, turn: props.turn}
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
        race,
        countryOrder,
        onCountryOrder: setCountryOrder,
        hallOfFame,
    }
    const missionsState = useMissions(props.missions, ownAccountId, allegiance?.countryId, accepted.listenForClicks)
    const [missionsDone, setMissionsDone] = useState<Mission[]>()
    usePaidMissions(missionsState, setMissionsDone)
    useFlagMissionDone(missionsState, (mission) => {
        if (allegiance) announce({kind: "flagMission", mission, country: allegiance.countryId})
    })
    const missions: MissionsPlaceProps | undefined = props.missions && {
        state: missionsState,
        onSignIn: guest ? openPitch ?? (() => openSheet("you")) : undefined,
    }
    const missionsNews = useSeenMissions((seenMissions) => missionsState.kind === "ready" ? unseen(missionsState.day.missions, seenMissions.seen) : 0)

    const you = {
        account: props.account,
        linkedMultiplier: clickBudget?.linkedMultiplier,
        playerInfo: props.playerInfo,
        listenForClicks: accepted.listenForClicks,
        clips: props.clips,
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
        support: props.support,
        pageContext: () => pageContextOf(window, graphics(), connection === "down"),
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

        {ready && !compact && battle && <BattleChip battle={battle}
                                                    season={season}
                                                    compact={false}/>}
        {ready && !compact && !battle && champion && <ChampionChip champion={champion} compact={false}/>}
        {ready && !compact && !battle && !champion && season && <SeasonChip season={season}
                                                                 nextBattle={nextBattle}
                                                                 finale={finale}
                                                                 dayEndsAt={dayEndsAt}
                                                                 compact={false}/>}

        {ready && compact && <StatusBar country={countryState}
                                        rank={rankOf(lines, countryState)}
                                        trailing={battle
                                            ? <BattleChip battle={battle}
                                                          season={season}
                                                          compact/>
                                            : champion
                                                ? <ChampionChip champion={champion} compact/>
                                                : season && <SeasonChip season={season}
                                                                        nextBattle={nextBattle}
                                                                        finale={finale}
                                                                        dayEndsAt={dayEndsAt}
                                                                        compact/>}/>}

        {ready && <ClickBudgetMeter budget={clickBudget}
                                    frozen={mood === "frozen" ? {nextSeason: schedule.seasons.next, country: countryState} : undefined}
                                    invite={invite}
                                    refusals={refusals}
                                    said={lockedOut && {text: heldLine(lockedOut.allegiance, Date.now()), id: lockedOut.id}}
                                    onSignIn={openPitch}
                                    compact={compact}
                                    popover={clicks}>
            {props.bonusListener && <Inventory charges={charges}
                                               rules={rules}
                                               switches={switches}
                                               onToggle={toggleSwitch}
                                               bombArmed={bombArmed}
                                               onToggleBomb={props.bomber ? toggleBomb : undefined}
                                               onUseRefill={spendRefill}
                                               onToggleShield={props.shielder ? () => toggleSwitch("shield") : undefined}
                                               notice={notice}/>}
        </ClickBudgetMeter>}

        {ready && crazyGames.chat && <ChatPanel
            backend={props.chatBackend}
            flagRoom={flagRoom}
            onSignIn={clickBudget?.linkedMultiplier ? openPitch : undefined}
            onLeftRoom={() => void props.allegiance?.refreshAllegiance().catch((e) => console.error("could not read the flag played for", e))}
            country={countryState}
            playSound={sound.play}
            accountId={ownAccountId}
            username={username}
            reporter={props.support}
            players={roster.kind === 'ready' ? roster.entries : undefined}
            compact={compact}
            open={compact ? sheet === "chat" : undefined}
            onOpenChange={compact ? (open) => open ? openSheet("chat") : closeSheet() : undefined}
            onUnread={setUnread}
            discord={discord}
            gifs={props.gifs}
        />}

        {ready && compact && <>
            {sheet === "board" && <BoardSheet {...board} onClose={closeSheet}/>}
            {sheet === "missions" && missions && <Sheet title="Missions" onClose={closeSheet}>
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
            {sheet === "season" && season && <Sheet title={`Season ${season.number}`} onClose={closeSheet}>
                <SeasonDetails season={season} nextBattle={nextBattle} finale={finale} dayEndsAt={dayEndsAt}/>
            </Sheet>}
            {sheet === "battle" && battle && <Sheet title="Battle" onClose={closeSheet}>
                <BattleDetails battle={battle} season={season}/>
            </Sheet>}
            {sheet === "clicks" && <Sheet title="Your clicks" onClose={closeSheet}>{clicks}</Sheet>}

            <TabBar chat={props.chatBackend !== undefined && crazyGames.chat}
                    invite={invite !== undefined}
                    unread={unread}
                    you={account.kind === 'ready' ? (linked ? "player" : "guest") : undefined}
                    missions={missions ? missionsNews : undefined}/>
        </>}

        {shot && <SharePreview shot={shot}
                               stats={stats}
                               onClose={discard}/>}

        {pitchOpen && guest && props.account && clickBudget?.linkedMultiplier && <SignInPitchModal
            state={account}
            store={props.account}
            multiplier={clickBudget.linkedMultiplier}
            onClose={() => setPitchOpen(false)}/>}

        {nudge.open && guest && props.account && clickBudget?.linkedMultiplier && <SignInNudgeModal
            state={account}
            store={props.account}
            multiplier={clickBudget.linkedMultiplier}
            fullBank={signInRefill}
            onClose={nudge.close}/>}

        {invitation.asked && allegiance && <InvitedModal key={invitation.asked.code}
                                                         friend={invitation.asked}
                                                         held={allegiance}
                                                         onSwitch={() => pickCountry(invitation.asked!)}
                                                         onClose={invitation.answer}/>}

        {tileHolder.holder && <HolderCard key={`holder-${tileHolder.holder.id}`}
                                                              holder={tileHolder.holder}
                                                              onHold={tileHolder.hold}
                                                              onClose={tileHolder.dismiss}/>}

        {territoryCard.shown && <TerritoryCard key={`territory-${territoryCard.shown.id}`}
                                               shown={territoryCard.shown}
                                               flag={countryState.code}
                                               players={territoryPlayers}
                                               onHold={territoryCard.hold}
                                               onClose={territoryCard.dismiss}/>}

        {props.playerInfo && <PlayerCardPlace backend={props.playerInfo}
                                              accountId={ownAccountId}
                                              reporter={props.support}/>}

        <Moments shown={ready}
                 battle={battle}
                 closed={closed}
                 finale={finale}
                 listenForClicks={accepted.listenForClicks}
                 clips={props.clips}
                 clipsAccount={linked && username !== undefined ? ownAccountId : undefined}
                 countryCode={countryState.code}
                 accountId={ownAccountId}
                 play={sound.play}
                 onWear={props.account?.wearTitle}/>

        {missionsDone && rules && <MissionsDone missions={missionsDone}
                                                rules={rules}
                                                onDone={() => setMissionsDone(undefined)}/>}

        {award && <BonusAward reward={award}
                              kept={firstWin}
                              onDone={() => {
                                  markWon(award.kind)
                                  dismissAward()
                              }}/>}

        {ready && connection === "down" && <ConnectionLost/>}

        {lastClose && newest === lastClose && <FortifyHint
            key={`close-${lastClose.id}`}
            close={lastClose.close}
            onDone={dismissClose}
        />}

        {lastFortified && newest === lastFortified && <FortifyNews
            key={`fortify-${lastFortified.id}`}
            fortified={lastFortified.fortified}
            onDone={dismissFortified}
        />}

        {fortifyCard && <FortifyCard fortified={fortifyCard} onDone={() => {
            fortifyGuide.markSeen()
            setFortifyCard(undefined)
        }}/>}

        {lastBomb && newest === lastBomb && <BombNews
            key={lastBomb.id}
            drop={lastBomb.drop}
            land={lastBomb.land}
            onDone={dismissBomb}
        />}

        {underAttack.moment && newest === underAttack.moment && <UnderAttack
            key={`attack-${underAttack.moment.at}`}
            flag={countryState.code}
            connect={discord.connect}
            goThere={flyToLastLoss}
            onDone={underAttack.dismiss}
        />}

        {battleOver.over && <BattleOver key={battleKeyOf(battleOver.over)}
                                        awaiting={battleOver.awaiting}
                                        onDone={battleOver.dismiss}/>}

        <Quiz/>

        {vpnBlocked && <VPNBlockedModal onClose={dismissVPNBlocked}/>}
        {bannedUntil !== undefined && <BannedModal until={bannedUntil} onClose={dismissBanned}/>}
        {account.kind === 'ready' && account.restored && props.account &&
            <WelcomeBackModal onClose={() => props.account?.welcomedBack()}/>}

        {sessionUnavailable && <SessionUnavailableModal onClose={dismissSessionUnavailable}/>}
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
