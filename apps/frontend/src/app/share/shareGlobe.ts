import {CapturedFrame} from "../viewer/capture.ts";
import {shareFileName, ShareStats, shareText} from "../../domain/shareCard.ts";
import {drawShareCard} from "./drawShareCard.ts";
import {deliverShare, ShareOutcome} from "./deliverShare.ts";

/** Capture, compose, deliver — the three halves of the button, in one place so
 *  the component holds nothing but the state the player can see. */
export async function shareGlobe(
    capture: () => Promise<CapturedFrame>,
    stats: ShareStats,
): Promise<ShareOutcome> {
    const frame = await capture()
    const image = await drawShareCard(frame, stats)
    const file = new File([image], shareFileName(stats.country.code), {type: "image/png"})

    return deliverShare(file, shareText(stats))
}
