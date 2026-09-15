// Downloads the US Navy Band's national anthem recordings (public domain, a work
// of the US government), encodes one per recording into static/anthems under a
// content-addressed name, and writes src/app/anthem/anthemsAsset.ts, which maps
// every country code that has an anthem to its file and its title.
//
// Needs ffmpeg on the PATH. Downloads are cached in node_modules/.cache/anthems.
import {execFileSync} from "node:child_process"
import crypto from "node:crypto"
import fs from "node:fs"
import path from "node:path"

const SOURCE = "https://archive.org/download/us-navy-band-national-anthems-public-domain/"
const CACHE = "node_modules/.cache/anthems"
const OUT = "static/anthems"
const ASSET = "src/app/anthem/anthemsAsset.ts"

// Country code → recording, as named in the archive. Where the band recorded a
// short and a complete version, the short one: it loops under a game, and the
// complete ones run to five minutes. Recordings under Removed/ are ones the band
// took off its site; they are used only where Current/ has nothing for a country.
const RECORDINGS = {
    af: "Current/Afghanistan", al: "Current/Albania", dz: "Current/Algeria", ao: "Current/Angola",
    ag: "Current/Antigua and Barbuda", ar: "Current/Argentina (Short)", am: "Current/Armenia",
    aw: "Current/Aruba", au: "Current/Australia (Complete)", at: "Current/Austria",
    az: "Current/Azerbaijan", bs: "Current/Bahamas", bh: "Current/Bahrain", bd: "Current/Bangladesh",
    bb: "Current/Barbados", by: "Current/Belarus", be: "Current/Belgium", bz: "Current/Belize",
    bj: "Current/Benin", bo: "Current/Bolivia", ba: "Current/Bosnia-Herzegovina",
    bw: "Current/Botswana", br: "Current/Brazil", bn: "Current/Brunei", bg: "Current/Bulgaria (Short)",
    bf: "Current/Burkina-Faso", kh: "Current/Cambodia", cm: "Current/Cameroon", ca: "Current/Canada",
    cv: "Current/Cape Verde", cf: "Current/Central African Republic", td: "Current/Chad",
    cl: "Current/Chile", cn: "Current/China", co: "Current/Colombia", km: "Current/Comoros",
    cg: "Current/Congo", ck: "Current/Cook Islands", cr: "Current/Costa Rica",
    ci: "Current/Cote d'Ivoire (Ivory Coast)", hr: "Current/Croatia", cu: "Current/Cuba",
    cz: "Current/Czech Republic", dk: "Current/Denmark (National Anthem)", dj: "Current/Djibouti",
    dm: "Current/Dominica (Commonwealth of)", do: "Current/Dominican Republic",
    tl: "Current/East Timor", ec: "Current/Ecuador", eg: "Current/Egypt", sv: "Current/El Salvador",
    er: "Current/Eritrea", ee: "Current/Estonia", et: "Current/Ethiopia", fj: "Current/Fiji",
    fi: "Current/Finland", fr: "Current/France", ga: "Current/Gabon", gm: "Current/Gambia",
    ge: "Current/Georgia", de: "Current/Germany", gh: "Current/Ghana", gr: "Current/Greece",
    gt: "Current/Guatemala", gw: "Current/Guinea-Bissau", gn: "Current/Guinea", gy: "Current/Guyana",
    ht: "Current/Haiti", hn: "Current/Honduras", hu: "Current/Hungary", is: "Current/Iceland",
    in: "Current/India", id: "Current/Indonesia", iq: "Current/Iraq", ie: "Current/Ireland",
    il: "Current/Israel", it: "Current/Italy (Short)", jm: "Current/Jamaica", jp: "Current/Japan",
    jo: "Current/Jordan", kz: "Current/Kazakhstan", ke: "Current/Kenya", kr: "Current/Korea, South",
    xk: "Current/Kosovo", kw: "Current/Kuwait", kg: "Current/Kyrgyzstan", la: "Current/Laos",
    lv: "Current/Latvia", lb: "Current/Lebanon", ls: "Current/Lesotho", lr: "Current/Liberia",
    ly: "Current/Libya", li: "Current/Liechtenstein", lt: "Current/Lithuania",
    lu: "Current/Luxembourg", mk: "Current/Macedonia", mg: "Current/Madagascar", mw: "Current/Malawi",
    my: "Current/Malaysia (short)", mv: "Current/Maldives", ml: "Current/Mali", mt: "Current/Malta",
    mh: "Current/Marshall Islands", mr: "Current/Mauritania", mu: "Current/Mauritius",
    mx: "Current/Mexico", fm: "Current/Micronesia (Short Version)", md: "Current/Moldova",
    mc: "Current/Monaco", me: "Current/Montenegro", ma: "Current/Morocco", mz: "Current/Mozambique",
    mm: "Current/Myanmar", na: "Current/Namibia", np: "Current/Nepal",
    cw: "Current/Netherlands Antilles", nl: "Current/Netherlands",
    nz: "Current/New Zealand (God Defend New Zealand)", ni: "Current/Nicaragua",
    ng: "Current/Nigeria", mp: "Current/Northern Mariana Islands", no: "Current/Norway",
    om: "Current/Oman", pk: "Current/Pakistan", pw: "Current/Palau (Belau)", pa: "Current/Panama",
    pg: "Current/Papua New Guinea", pe: "Current/Peru", ph: "Current/Philippines", pl: "Current/Poland",
    pt: "Current/Portugal", pr: "Current/Puerto Rico", qa: "Current/Qatar", ro: "Current/Romania",
    ru: "Current/Russia", rw: "Current/Rwanda", st: "Current/Sao Tome and Principe",
    sa: "Current/Saudi Arabia", sn: "Current/Senegal", rs: "Current/Serbia",
    sc: "Current/Seychelles", sl: "Current/Sierra Leone", sg: "Current/Singapore",
    sk: "Current/Slovak Republic", si: "Current/Slovenia", so: "Current/Somalia",
    za: "Current/South Africa", ss: "Current/South Sudan", es: "Current/Spain (Short)",
    lk: "Current/Sri Lanka", kn: "Current/St. Kitt's and Nevis", sd: "Current/Sudan",
    sz: "Current/Swaziland", se: "Current/Sweden", ch: "Current/Switzerland",
    tj: "Current/Tajikistan", tz: "Current/Tanzania", tg: "Current/Togo",
    tt: "Current/Trinidad and Tobago", tn: "Current/Tunisia", tr: "Current/Turkey",
    tm: "Current/Turkmenistan", ug: "Current/Uganda", ua: "Current/Ukraine",
    ae: "Current/United Arab Emirates", gb: "Current/United Kingdom", us: "Current/United States",
    uy: "Current/Uruguay (Short)", uz: "Current/Uzbekistan", vu: "Current/Vanuatu",
    va: "Current/Vatican City (Holy See)", ve: "Current/Venezuela", vn: "Current/Vietnam",
    vi: "Current/Virgin Islands", ye: "Current/Yemen",

    gq: "Removed/Equatorial Guinea (2019)", ir: "Removed/Iran (2000)", ki: "Removed/Kiribati (2019)",
    mn: "Removed/Mongolia (2019)", py: "Removed/Paraguay (2000)", lc: "Removed/St. Lucia (2019)",
    vc: "Removed/St. Vincent (2019)", sr: "Removed/Suriname (2019)", sy: "Removed/Syria (2000)",
    th: "Removed/Thailand (National Anthem) (2000)",
}

// What each recording is called, as the player shows it: the name the anthem is
// known by, in its own language where that is how the world knows it.
const TITLES = {
    af: "Milli Surood", al: "Himni i Flamurit", dz: "Kassaman", ao: "Angola Avante",
    ag: "Fair Antigua, We Salute Thee", ar: "Himno Nacional Argentino", am: "Mer Hayrenik",
    aw: "Aruba Dushi Tera", au: "Advance Australia Fair", at: "Land der Berge, Land am Strome",
    az: "Azərbaycan marşı", bs: "March On, Bahamaland", bh: "Bahrainona", bd: "Amar Sonar Bangla",
    bb: "In Plenty and In Time of Need", by: "My, Belarusy", be: "La Brabançonne", bz: "Land of the Free",
    bj: "L'Aube Nouvelle", bo: "Bolivianos, el Hado Propicio", ba: "Intermeco", bw: "Fatshe leno la rona",
    br: "Hino Nacional Brasileiro", bn: "Allah Peliharakan Sultan", bg: "Mila Rodino",
    bf: "Une Seule Nuit", kh: "Nokor Reach", cm: "O Cameroon, Cradle of Our Forefathers", ca: "O Canada",
    cv: "Cântico da Liberdade", cf: "La Renaissance", td: "La Tchadienne", cl: "Himno Nacional de Chile",
    cn: "March of the Volunteers", co: "¡Oh, Gloria Inmarcesible!", km: "Udzima wa ya Masiwa",
    cg: "La Congolaise", ck: "Te Atua Mou E", cr: "Noble patria, tu hermosa bandera",
    ci: "L'Abidjanaise", hr: "Lijepa naša domovino", cu: "La Bayamesa", cz: "Kde domov můj",
    dk: "Der er et yndigt land", dj: "Djibouti", dm: "Isle of Beauty, Isle of Splendour",
    do: "Quisqueyanos valientes", tl: "Pátria", ec: "Salve, Oh Patria", eg: "Bilady, Bilady, Bilady",
    sv: "Saludemos la Patria orgullosos", er: "Ertra, Ertra, Ertra", ee: "Mu isamaa, mu õnn ja rõõm",
    et: "March Forward, Dear Mother Ethiopia", fj: "God Bless Fiji", fi: "Maamme", fr: "La Marseillaise",
    ga: "La Concorde", gm: "For The Gambia, Our Homeland", ge: "Tavisupleba", de: "Das Lied der Deutschen",
    gh: "God Bless Our Homeland Ghana", gr: "Hymn to Liberty", gt: "Guatemala Feliz",
    gw: "Esta é a Nossa Pátria Bem Amada", gn: "Liberté", gy: "Dear Land of Guyana, of Rivers and Plains",
    ht: "La Dessalinienne", hn: "Tu bandera es un lampo de cielo", hu: "Himnusz", is: "Lofsöngur",
    in: "Jana Gana Mana", id: "Indonesia Raya", iq: "Mawtini", ie: "Amhrán na bhFiann", il: "Hatikvah",
    it: "Il Canto degli Italiani", jm: "Jamaica, Land We Love", jp: "Kimigayo",
    jo: "As-Salam al-Malaki al-Urduni", kz: "Menıñ Qazaqstanym", ke: "Ee Mungu Nguvu Yetu",
    kr: "Aegukga", xk: "Europe", kw: "Al-Nashid Al-Watani", kg: "State Anthem of the Kyrgyz Republic",
    la: "Pheng Xat Lao", lv: "Dievs, svētī Latviju!", lb: "Kulluna lil-watan",
    ls: "Lesotho Fatše La Bo-ntata Rona", lr: "All Hail, Liberia, Hail!", ly: "Libya, Libya, Libya",
    li: "Oben am jungen Rhein", lt: "Tautiška giesmė", lu: "Ons Heemecht", mk: "Denes nad Makedonija",
    mg: "Ry Tanindrazanay malala ô!", mw: "Mulungu dalitsa Malaŵi", my: "Negaraku", mv: "Gaumii salaam",
    ml: "Pour l'Afrique et pour toi, Mali", mt: "L-Innu Malti", mh: "Forever Marshall Islands",
    mr: "Country of the Proud, Guiding Noblemen", mu: "Motherland", mx: "Himno Nacional Mexicano",
    fm: "Patriots of Micronesia", md: "Limba noastră", mc: "Hymne Monégasque",
    me: "Oj, svijetla majska zoro", ma: "Cherifian Anthem", mz: "Pátria Amada", mm: "Kaba Ma Kyei",
    na: "Namibia, Land of the Brave", np: "Sayaun Thunga Phulka", cw: "Anthem of the Netherlands Antilles",
    nl: "Wilhelmus", nz: "God Defend New Zealand", ni: "Salve a ti, Nicaragua",
    ng: "Arise, O Compatriots", mp: "Gi Talo Gi Halom Tasi", no: "Ja, vi elsker dette landet",
    om: "As-Salam as-Sultani", pk: "Qaumi Taranah", pw: "Belau rekid", pa: "Himno Istmeño",
    pg: "O Arise, All You Sons", pe: "Himno Nacional del Perú", ph: "Lupang Hinirang",
    pl: "Mazurek Dąbrowskiego", pt: "A Portuguesa", pr: "La Borinqueña", qa: "As Salam al Amiri",
    ro: "Deșteaptă-te, române!", ru: "State Anthem of the Russian Federation", rw: "Rwanda Nziza",
    st: "Independência total", sa: "Aash Al Maleek", sn: "Le Lion rouge", rs: "Bože pravde",
    sc: "Koste Seselwa", sl: "High We Exalt Thee, Realm of the Free", sg: "Majulah Singapura",
    sk: "Nad Tatrou sa blýska", si: "Zdravljica", so: "Qolobaa Calankeed",
    za: "National Anthem of South Africa", ss: "South Sudan Oyee!", es: "Marcha Real",
    lk: "Sri Lanka Matha", kn: "O Land of Beauty!", sd: "Nahnu Jund Allah Jund Al-watan",
    sz: "Nkulunkulu Mnikati wetibusiso temaSwati", se: "Du gamla, du fria", ch: "Swiss Psalm",
    tj: "Surudi Milli", tz: "Mungu Ibariki Afrika", tg: "Terre de nos aïeux",
    tt: "Forged from the Love of Liberty", tn: "Humat al-Hima", tr: "İstiklâl Marşı",
    tm: "State Anthem of Turkmenistan", ug: "Oh Uganda, Land of Beauty", ua: "Shche ne vmerla Ukrainy",
    ae: "Ishy Bilady", gb: "God Save the King", us: "The Star-Spangled Banner",
    uy: "Orientales, la Patria o la tumba", uz: "State Anthem of Uzbekistan", vu: "Yumi, Yumi, Yumi",
    va: "Inno e Marcia Pontificale", ve: "Gloria al Bravo Pueblo", vn: "Tiến Quân Ca",
    vi: "Virgin Islands March", ye: "United Republic",

    gq: "Caminemos pisando las sendas de nuestra inmensa felicidad",
    ir: "National Anthem of the Islamic Republic of Iran", ki: "Teirake Kaini Kiribati",
    mn: "Mongol Ulsyn Töriin Duulal", py: "Paraguayos, República o Muerte",
    lc: "Sons and Daughters of Saint Lucia", vc: "Saint Vincent, Land So Beautiful",
    sr: "God zij met ons Suriname", sy: "Humat ad-Diyar", th: "Phleng Chat Thai",
}

const untitled = Object.keys(RECORDINGS).filter((code) => !TITLES[code])
if (untitled.length) throw new Error(`No title for: ${untitled.join(", ")}`)

// Places on the map whose anthem is another country's: territories, and the
// home nations that stand to God Save the King. They share that country's file.
const SHARES = {
    gb: ["gb-eng", "gb-nir", "ai", "bm", "io", "ky", "fk", "gi", "gg", "im", "je", "ms", "pn", "sh", "tc", "vg", "gs"],
    fr: ["gf", "pf", "tf", "gp", "mq", "yt", "nc", "re", "bl", "mf", "pm", "wf"],
    us: ["as", "gu", "um"],
    au: ["cx", "cc", "hm", "nf"],
    nz: ["nu", "tk"],
    dk: ["fo", "gl"],
    no: ["bv", "sj"],
    nl: ["bq"],
    cw: ["sx"],
    fi: ["ax"],
    cn: ["hk", "mo"],
}

// Every anthem at one loudness, so the music does not jump when the lead
// changes, and the silence at either end trimmed, so the loop has no gap.
const FILTER = [
    "silenceremove=start_periods=1:start_threshold=-50dB",
    "areverse",
    "silenceremove=start_periods=1:start_threshold=-50dB",
    "areverse",
    "loudnorm=I=-18:TP=-2:LRA=11",
].join(",")

async function download(recording) {
    const cached = path.join(CACHE, recording + ".mp3")
    if (fs.existsSync(cached)) return cached

    const response = await fetch(SOURCE + encodeURIComponent(recording + ".mp3").replace(/%2F/g, "/"))
    if (!response.ok) throw new Error(`${recording}: HTTP ${response.status}`)
    fs.mkdirSync(path.dirname(cached), {recursive: true})
    fs.writeFileSync(cached, Buffer.from(await response.arrayBuffer()))
    return cached
}

function encode(input) {
    const tmp = path.join(CACHE, "encoding.m4a")
    execFileSync("ffmpeg", [
        "-v", "error", "-y", "-i", input,
        "-af", FILTER, "-ar", "44100", "-ac", "1",
        "-c:a", "aac", "-b:a", "56k",
        "-map_metadata", "-1", "-fflags", "+bitexact", "-flags:a", "+bitexact",
        "-movflags", "+faststart",
        tmp,
    ])
    return fs.readFileSync(tmp)
}

// The archive is slow one file at a time, so the downloads run side by side.
const queue = [...new Set(Object.values(RECORDINGS))]
const cached = new Map()
await Promise.all(Array.from({length: 8}, async () => {
    for (let recording = queue.pop(); recording; recording = queue.pop()) {
        cached.set(recording, await download(recording))
    }
}))

fs.rmSync(OUT, {recursive: true, force: true})
fs.mkdirSync(OUT, {recursive: true})

const anthems = {}
for (const [code, recording] of Object.entries(RECORDINGS)) {
    const bytes = encode(cached.get(recording))
    const hash = crypto.createHash("sha256").update(bytes).digest("hex").slice(0, 8)
    const file = `${code}-${hash}.m4a`
    fs.writeFileSync(path.join(OUT, file), bytes)
    anthems[code] = {url: `/static/anthems/${file}`, title: TITLES[code]}
    console.log(`${code}  ${recording}  ${Math.round(bytes.length / 1024)}K`)
}
for (const [owner, codes] of Object.entries(SHARES)) {
    for (const code of codes) anthems[code] = anthems[owner]
}

const sorted = Object.keys(anthems).sort().map((code) =>
    `    ${JSON.stringify(code)}: {url: ${JSON.stringify(anthems[code].url)}, title: ${JSON.stringify(anthems[code].title)}},`)
fs.writeFileSync(ASSET, [
    "// Generated by `npm run anthems` — do not edit.",
    "",
    "/** Country code → its national anthem, played by the US Navy Band (public domain). */",
    "export const ANTHEMS: Readonly<Record<string, {url: string, title: string}>> = {",
    ...sorted,
    "}",
    "",
].join("\n"))

const total = fs.readdirSync(OUT).reduce((sum, file) => sum + fs.statSync(path.join(OUT, file)).size, 0)
console.log(`${Object.keys(RECORDINGS).length} recordings, ${Object.keys(anthems).length} countries, ${(total / 1e6).toFixed(1)} MB`)
