import {CapturedFrame} from "../viewer/capture.ts";
import {shareFileName, ShareStats} from "../../domain/shareCard.ts";
import {drawShareCard} from "./drawShareCard.ts";

/** Capture and compose, in one call — everything up to holding the picture.
 *  What happens to it afterwards is the player's to choose; see deliverShare. */
export async function takePicture(
    capture: () => Promise<CapturedFrame>,
    stats: ShareStats,
): Promise<File> {
    const frame = await capture()
    const image = await drawShareCard(frame, stats)

    return new File([image], shareFileName(stats.country.code), {type: "image/png"})
}
