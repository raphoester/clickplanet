import {PromiseClient} from "@connectrpc/connect"
import {
    alertsAdmin, antibotAdmin, authAdmin, bonusesAdmin, boxesAdmin, chatAdmin, clipsAdmin, flagsAdmin, gamemasterAdmin, missionsAdmin, planetAdmin, playerAdmin,
    progressionAdmin, promoAdmin, publishingAdmin, quizzesAdmin, supportAdmin, vandalsquadAdmin,
} from "./api.ts"
import {AdminService as AlertsAdminService} from "./gen/grpc/alerts/v1/admin_connect.ts"
import {AdminService as AntibotAdminService} from "./gen/grpc/antibot/v1/admin_connect.ts"
import {AdminService as AuthAdminService} from "./gen/grpc/auth/v1/admin_connect.ts"
import {AdminService as BonusesAdminService} from "./gen/grpc/bonuses/v1/admin_connect.ts"
import {AdminService as BoxesAdminService} from "./gen/grpc/boxes/v1/admin_connect.ts"
import {AdminService as ChatAdminService} from "./gen/grpc/chat/v1/admin_connect.ts"
import {AdminService as ClipsAdminService} from "./gen/grpc/clips/v1/admin_connect.ts"
import {AdminService as FlagsAdminService} from "./gen/grpc/flags/v1/admin_connect.ts"
import {AdminService as PlanetAdminService} from "./gen/grpc/planet/v1/admin_connect.ts"
import {AdminService as PlayerAdminService} from "./gen/grpc/player/v1/admin_connect.ts"
import {AdminService as ProgressionAdminService} from "./gen/grpc/progression/v1/admin_connect.ts"
import {AdminService as GamemasterAdminService} from "./gen/grpc/gamemaster/v1/admin_connect.ts"
import {AdminService as MissionsAdminService} from "./gen/grpc/missions/v1/admin_connect.ts"
import {AdminService as PromoAdminService} from "./gen/grpc/promo/v1/admin_connect.ts"
import {AdminService as PublishingAdminService} from "./gen/grpc/publishing/v1/admin_connect.ts"
import {AdminService as QuizzesAdminService} from "./gen/grpc/quizzes/v1/admin_connect.ts"
import {AdminService as SupportAdminService} from "./gen/grpc/support/v1/admin_connect.ts"
import {AdminService as VandalsquadAdminService} from "./gen/grpc/vandalsquad/v1/admin_connect.ts"

// Each page takes the services it calls as these, so a test hands it FakeAdmin (fakeAdmin.ts) instead.
export type ChatAdmin = Pick<
    PromiseClient<typeof ChatAdminService>,
    "getMessage" | "listMessages" | "removeMessage" | "mute" | "unmute" | "getMute" | "listModeration"
>
export type AntibotAdmin = PromiseClient<typeof AntibotAdminService>
export type AlertsAdmin = PromiseClient<typeof AlertsAdminService>
// The rule sets of what a box caught or a right answer grants, and the sign-in refill.
export type BonusesAdmin = PromiseClient<typeof BonusesAdminService>
// The rule sets the boxes are offered by.
export type BoxesAdmin = PromiseClient<typeof BoxesAdminService>
// The rule sets the quizzes are asked by, and the bank's questions.
export type QuizzesAdmin = PromiseClient<typeof QuizzesAdminService>
export type PlanetAdmin = PromiseClient<typeof PlanetAdminService>
export type PlayerAdmin = PromiseClient<typeof PlayerAdminService>
export type ProgressionAdmin = PromiseClient<typeof ProgressionAdminService>
export type GamemasterAdmin = PromiseClient<typeof GamemasterAdminService>
// The numbers missions draws with. The flag days and totals are the connector's reads.
export type MissionsAdmin = Pick<PromiseClient<typeof MissionsAdminService>, "getRules" | "saveRules">
export type SupportAdmin = Pick<
    PromiseClient<typeof SupportAdminService>,
    "listNotes" | "answerNote" | "closeNote" | "listReports" | "decideReport" | "writeLetter" | "listLetters" | "closeLetter"
>

// The videos the operator posts by hand.
export type ClipsAdmin = Pick<PromiseClient<typeof ClipsAdminService>, "createVideoUpload" | "keepVideo" | "listVideos" | "removeVideo">

// Where the accounts came from, counted. Its sign-in and connection calls are App's and the Connect page's.
export type AuthAdmin = Pick<PromiseClient<typeof AuthAdminService>, "countArrivals">

// The posts' numbers, a snapshot a day the server takes.
export type PublishingAdmin = Pick<PromiseClient<typeof PublishingAdminService>, "listSnapshots" | "listRows">

// The battle teasers: the background library and each rule set's cards. The teasers job's calls are not the page's.
export type PromoAdmin = Pick<
    PromiseClient<typeof PromoAdminService>,
    "createBackgroundUpload" | "keepBackground" | "listBackgrounds" | "removeBackground" | "getCards" | "saveCards"
>

// The swastikas the vandal squad flagged on the map.
export type VandalsquadAdmin = PromiseClient<typeof VandalsquadAdminService>

// The flags the game ships with and the ones opened since, which name a code the page does not.
export type FlagsAdmin = PromiseClient<typeof FlagsAdminService>

export type Clients = {
    alerts: AlertsAdmin, antibot: AntibotAdmin, auth: AuthAdmin, bonuses: BonusesAdmin, boxes: BoxesAdmin, chat: ChatAdmin, planet: PlanetAdmin, player: PlayerAdmin,
    progression: ProgressionAdmin, gamemaster: GamemasterAdmin, missions: MissionsAdmin, support: SupportAdmin, clips: ClipsAdmin,
    promo: PromoAdmin, publishing: PublishingAdmin, quizzes: QuizzesAdmin, vandalsquad: VandalsquadAdmin, flags: FlagsAdmin,
}

export const CLIENTS: Clients = {
    alerts: alertsAdmin, antibot: antibotAdmin, auth: authAdmin, bonuses: bonusesAdmin, boxes: boxesAdmin, chat: chatAdmin, planet: planetAdmin, player: playerAdmin,
    progression: progressionAdmin, gamemaster: gamemasterAdmin, missions: missionsAdmin, support: supportAdmin, clips: clipsAdmin,
    promo: promoAdmin, publishing: publishingAdmin, quizzes: quizzesAdmin, vandalsquad: vandalsquadAdmin, flags: flagsAdmin,
}
