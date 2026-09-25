// Regenerates golden.json from cases.json by running the widget's own card code:
// node golden.mjs ~/Projects/ii-widget-statusphere
import { readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const widget = process.argv[2];
if (!widget) {
    console.error("usage: node golden.mjs <ii-widget-statusphere dir>");
    process.exit(2);
}
const here = dirname(fileURLToPath(import.meta.url));

const layoutSource = readFileSync(join(widget, "CardLayouts.js"), "utf8").replace(/^\.pragma library\s*$/m, "");
const exported = ["tileTypes", "typeOf", "formOf", "pack", "fallbackDetail", "withKnownBackground", "sizes", "shapes", "pictureUrlOf", "layoutKey", "colorRoles", "rowRows", "detailRows"];
const CardLayouts = new Function(`${layoutSource}\nreturn { ${exported.join(", ")} };`)();

String.prototype.arg = function (value) {
    const markers = [...this.matchAll(/%(\d)/g)].map(m => Number(m[1]));
    return markers.length === 0 ? String(this) : this.replaceAll(`%${Math.min(...markers)}`, String(value));
};
const Translation = { tr: s => s };

const qml = readFileSync(join(widget, "Statusphere.qml"), "utf8");

function block(from) {
    const open = qml.indexOf("{", from);
    let depth = 0;
    for (let i = open; i < qml.length; i++) {
        if (qml[i] === "{")
            depth++;
        else if (qml[i] === "}" && --depth === 0)
            return qml.slice(open, i + 1);
    }
    throw new Error(`unbalanced block at ${from}`);
}

function qmlFunction(name) {
    const at = qml.search(new RegExp(`function ${name}\\(`));
    if (at < 0)
        throw new Error(`no function ${name} in Statusphere.qml`);
    const params = qml.slice(qml.indexOf("(", at) + 1, qml.indexOf(")", at)).replace(/:\s*\w+/g, "");
    return new Function("root", "CardLayouts", "Translation", `return function (${params}) ${block(at)};`);
}

function qmlBinding(name) {
    const at = qml.search(new RegExp(`property var ${name}:`));
    if (at < 0)
        throw new Error(`no property ${name} in Statusphere.qml`);
    return new Function("root", "CardLayouts", "Translation", `return (function () ${block(at)})();`);
}

const functions = ["deviceRank", "stalled", "compareDevices", "labelDevice", "trackKey", "musicDevices", "gameDevices", "videoDevices", "alarmDevices", "meetingDevices", "currentPhotoFor", "percentForField", "iconForField", "formatUptime", "systemFieldsFor", "fieldsFor", "labelForKey", "detailFieldsFor", "fieldFor", "layoutFor", "ownsSurface", "sanitizeTile", "expandWildcardTiles", "surfaceTiles", "deviceForTile", "tileHasData", "hiddenFor"];

function roomFor(input) {
    const root = {
        "selfDeviceId": "",
        "selfAccountId": "",
        "hiding": false,
        "staleGap": 45,
        "stalledDeviceIds": [],
        "_now": Date.parse("2026-09-25T12:00:00Z"),
        "nativeFieldKeys": ["cpu", "mem", "ram", "memory", "disk", "load", "uptime", "workspace", "active_app", "active_window", "package_count"],
        "members": input.members,
        "photos": input.photos ?? []
    };
    for (const name of functions)
        root[name] = qmlFunction(name)(root, CardLayouts, Translation);
    root.photosByAccountId = qmlBinding("photosByAccountId")(root, CardLayouts, Translation);
    root.accountsById = qmlBinding("accountsById")(root, CardLayouts, Translation);
    return root;
}

function formNameOf(t) {
    const type = CardLayouts.typeOf(t);
    const form = CardLayouts.formOf(t);
    return Object.keys(type?.forms ?? {}).find(name => type.forms[name] === form) ?? "";
}

function tileOut(root, account, p) {
    const t = p.tile;
    const type = CardLayouts.typeOf(t);
    const hasData = root.tileHasData(account, t);
    const out = {
        "col": p.col,
        "row": p.row,
        "cols": p.cols,
        "rows": p.rows,
        "type": t.type,
        "form": formNameOf(t) || undefined,
        "color": t.color in CardLayouts.colorRoles ? t.color : undefined,
        "dimmed": !hasData && t.onMissing === "dim" ? true : undefined
    };
    if (type.needsField) {
        const field = root.fieldFor(root.deviceForTile(account, t), t.field);
        Object.assign(out, {
            "field": t.field,
            "label": field?.label ?? root.labelForKey(t.field),
            "value": field?.value || undefined,
            "note": field?.note || undefined,
            "icon": field?.icon || undefined,
            "percent": field?.percent ?? undefined
        });
    } else if (type.reads === "music") {
        const d = root.musicDevices(account)[0];
        Object.assign(out, {
            "title": d?.spotify_track || d?.spotify_display || undefined,
            "subtitle": d?.spotify_artist || undefined,
            "image_url": d?.spotify_art_url || undefined
        });
    } else if (type.reads === "game") {
        const d = root.gameDevices(account)[0];
        Object.assign(out, {
            "title": d?.game_display || d?.game_name || undefined,
            "image_url": d?.game_hero_url || d?.game_header_url || undefined
        });
    } else if (type.reads === "video") {
        const d = root.videoDevices(account)[0];
        Object.assign(out, {
            "title": d?.video_title || undefined,
            "subtitle": d?.video_channel || undefined,
            "icon": d ? (d.video_status === "paused" ? "pause" : "play_arrow") : undefined,
            "percent": d?.video_length > 0 ? (d.video_position ?? 0) / d.video_length * 100 : undefined,
            "value": d?.video_length > 0 ? String(d.video_length) : undefined
        });
    } else if (type.reads === "alarm") {
        const d = root.alarmDevices(account)[0];
        out.value = d?.alarm_at !== undefined ? String(d.alarm_at) : undefined;
    } else if (type.reads === "meeting") {
        const d = root.meetingDevices(account)[0];
        out.value = d?.meeting_until !== undefined ? String(d.meeting_until) : undefined;
    } else if (type.art === "photo") {
        out.image_url = root.currentPhotoFor(account)?.path || undefined;
    } else if (type.art === "picture") {
        out.image_url = CardLayouts.pictureUrlOf(t) || undefined;
    }
    return out;
}

function placedOut(root, account, tiles, maxRows) {
    const shown = tiles.filter(t => t.onMissing !== "hide" || root.tileHasData(account, t));
    return CardLayouts.pack(shown, maxRows).map(p => tileOut(root, account, p));
}

function cardsFor(input) {
    const root = roomFor(input);
    return Object.values(root.accountsById).map(account => {
        const hidden = root.hiddenFor(account);
        const ownsRow = !hidden && root.ownsSurface(account, "row");
        return {
            "account_id": account.id,
            "row": ownsRow ? placedOut(root, account, root.surfaceTiles(account, "row"), CardLayouts.rowRows) : null,
            "detail": hidden ? [] : placedOut(root, account, root.surfaceTiles(account, "detail"), CardLayouts.detailRows)
        };
    });
}

const cases = JSON.parse(readFileSync(join(here, "cases.json"), "utf8"));
const golden = {};
for (const [name, input] of Object.entries(cases))
    golden[name] = cardsFor(input);
writeFileSync(join(here, "golden.json"), JSON.stringify(golden, null, 2) + "\n");
