package app.statusphere

import androidx.annotation.StringRes
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AssistChip
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.PrimaryTabRow
import androidx.compose.material3.Surface
import androidx.compose.material3.Tab
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import app.statusphere.mobile.Mobile
import org.json.JSONArray
import org.json.JSONObject

data class TileKind(val kind: String, val forms: List<String>, val sizes: List<String>)

data class CustomTile(val kind: String, val form: String, val size: String)

data class CustomTiles(val kinds: List<TileKind>, val chosen: List<CustomTile>)

fun tileKindsOf(json: String): List<TileKind> = JSONArray(json).let { kinds ->
    List(kinds.length()) {
        val k = kinds.getJSONObject(it)
        TileKind(k.getString("kind"), k.getJSONArray("forms").strings(), k.getJSONArray("sizes").strings())
    }
}

fun customTilesOf(json: String): List<CustomTile> = JSONArray(json).let { tiles ->
    List(tiles.length()) {
        val t = tiles.getJSONObject(it)
        CustomTile(t.getString("kind"), t.getString("form"), t.getString("size"))
    }
}

data class TileFit(val placed: Boolean, val sizes: List<String>)

data class CustomFit(val tiles: List<TileFit>, val room: List<String>)

data class CustomPreview(val tiles: List<CustomTile>, val card: Card, val fit: CustomFit)

fun customFitOf(json: String): CustomFit = JSONObject(json).let { fit ->
    val tiles = fit.getJSONArray("tiles")
    CustomFit(
        List(tiles.length()) { tiles.getJSONObject(it).let { t -> TileFit(t.getBoolean("placed"), t.optJSONArray("sizes")?.strings().orEmpty()) } },
        fit.optJSONArray("room")?.strings().orEmpty(),
    )
}

fun List<CustomTile>.toJSON(): String =
    JSONArray(map { JSONObject().put("kind", it.kind).put("form", it.form).put("size", it.size) }).toString()

private enum class CardSurface(val wire: String, @StringRes val title: Int) {
    ROW(Mobile.RowSurface, R.string.appearance_row),
    DETAIL(Mobile.DetailSurface, R.string.appearance_detail),
}

private val KIND_NAMES = mapOf(
    "music" to R.string.tile_kind_music,
    "video" to R.string.tile_kind_video,
    "app" to R.string.tile_kind_app,
    "battery" to R.string.tile_kind_battery,
    "alarm" to R.string.tile_alarm,
    "meeting" to R.string.tile_kind_meeting,
)

private val FORM_NAMES = mapOf(
    "cover" to R.string.form_cover,
    "vinyl" to R.string.form_vinyl,
    "wave" to R.string.form_wave,
    "text" to R.string.form_text,
    "big" to R.string.form_big,
    "bar" to R.string.form_bar,
    "ring" to R.string.form_ring,
    "number" to R.string.form_number,
)

private const val NEW_TILE_SIZE = "2x1"

private const val GLYPH_COLUMNS = 4
private const val GLYPH_ROWS = 2
private val SizeGlyphCell = 9.dp
private val SizeGlyphGap = 2.dp
private const val DISABLED_ALPHA = 0.38f

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AppearanceSheet(onDismiss: () -> Unit) {
    var surface by rememberSaveable { mutableStateOf(CardSurface.ROW) }
    var error by remember { mutableStateOf<String?>(null) }

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)) {
        PrimaryTabRow(selectedTabIndex = surface.ordinal) {
            CardSurface.entries.forEach {
                Tab(selected = it == surface, onClick = { surface = it }, text = { Text(stringResource(it.title)) })
            }
        }
        Column(
            Modifier.verticalScroll(rememberScrollState()).padding(CardPadding),
            verticalArrangement = Arrangement.spacedBy(CardSpacing),
        ) {
            error?.let {
                Text(
                    stringResource(R.string.appearance_failed, it),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.error,
                )
            }
            key(surface) { CustomEditor(surface, onError = { error = it }) }
        }
    }
}

@Composable
private fun SurfacePreview(surface: CardSurface, card: Card) {
    val tiles = if (surface == CardSurface.ROW) card.row else card.detail
    when {
        tiles == null -> PackNote(R.string.pack_default_row)
        tiles.isEmpty() && surface == CardSurface.ROW -> PackNote(R.string.pack_header_only)
        tiles.isEmpty() -> PackNote(R.string.pack_nothing_yet)
        else -> TileGrid(tiles)
    }
}

@Composable
private fun CustomEditor(surface: CardSurface, onError: (String) -> Unit) {
    var kinds by remember { mutableStateOf<List<TileKind>>(emptyList()) }
    var chosen by remember { mutableStateOf<List<CustomTile>?>(null) }
    var preview by remember { mutableStateOf<CustomPreview?>(null) }
    var selected by remember { mutableStateOf<Int?>(null) }

    LaunchedEffect(surface) {
        PresenceService.customTiles(surface.wire)
            .onSuccess { kinds = it.kinds; chosen = it.chosen }
            .onFailure { onError(it.message ?: it.javaClass.simpleName) }
    }
    LaunchedEffect(chosen) {
        val tiles = chosen ?: return@LaunchedEffect
        PresenceService.previewCustom(surface.wire, tiles)
            .onSuccess { preview = it }
            .onFailure { onError(it.message ?: it.javaClass.simpleName) }
    }

    fun change(next: List<CustomTile>, select: Int?) {
        chosen = next
        selected = select
        PresenceService.saveCustomSoon(surface.wire, next)
    }

    val shown = preview ?: return
    val current = shown.tiles
    val placed = shown.fit.tiles.indices.filter { shown.fit.tiles[it].placed }
    val tiles = if (surface == CardSurface.ROW) shown.card.row.orEmpty() else shown.card.detail

    when {
        current.isEmpty() && surface == CardSurface.DETAIL -> PackNote(R.string.pack_custom_empty_detail)
        current.isEmpty() -> PackNote(R.string.pack_header_only)
        tiles.size == placed.size -> TileGrid(
            tiles,
            selected = placed.indexOf(selected).takeIf { it >= 0 },
            onTileClick = { selected = placed[it].takeIf { i -> i != selected } },
        )
        else -> SurfacePreview(surface, shown.card)
    }
    val leftOut = current.indices - placed.toSet()
    if (leftOut.isNotEmpty()) {
        FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(
                stringResource(R.string.pack_custom_left_out),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.error,
                modifier = Modifier.align(Alignment.CenterVertically),
            )
            leftOut.forEach { i ->
                FilterChip(selected = i == selected, onClick = { selected = i }, label = { Text(kindName(current[i].kind)) })
            }
        }
    }

    val index = selected?.takeIf { it in current.indices }
    val kind = index?.let { i -> kinds.find { it.kind == current[i].kind } }
    if (index != null && kind != null) {
        TilePanel(
            kind,
            current[index],
            fits = shown.fit.tiles[index].sizes,
            placed = index in placed,
            canMoveEarlier = index > 0,
            canMoveLater = index < current.lastIndex,
            onChange = { tile -> change(current.toMutableList().also { it[index] = tile }, index) },
            onMove = { by -> change(current.toMutableList().also { it.add(index + by, it.removeAt(index)) }, index + by) },
            onRemove = { change(current.filterIndexed { i, _ -> i != index }, null) },
        )
    } else if (current.isNotEmpty()) {
        PackNote(R.string.pack_custom_tap_hint)
    }

    val missing = kinds.filter { k -> current.none { it.kind == k.kind } }
    if (missing.isEmpty()) return
    Text(stringResource(R.string.pack_custom_add), style = MaterialTheme.typography.titleSmall)
    val room = shown.fit.room
    if (room.isEmpty()) PackNote(R.string.pack_custom_no_room)
    FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        missing.forEach { k ->
            AssistChip(
                enabled = room.isNotEmpty(),
                onClick = {
                    val size = NEW_TILE_SIZE.takeIf { it in room } ?: room.first()
                    change(current + CustomTile(k.kind, k.forms.first(), size), current.size)
                },
                label = { Text(stringResource(R.string.pack_custom_add_kind, kindName(k.kind))) },
            )
        }
    }
}

@Composable
private fun kindName(kind: String): String = KIND_NAMES[kind]?.let { stringResource(it) } ?: kind

@Composable
private fun TilePanel(
    kind: TileKind,
    tile: CustomTile,
    fits: List<String>,
    placed: Boolean,
    canMoveEarlier: Boolean,
    canMoveLater: Boolean,
    onChange: (CustomTile) -> Unit,
    onMove: (Int) -> Unit,
    onRemove: () -> Unit,
) {
    val colors = MaterialTheme.colorScheme
    Surface(shape = CardShape, color = colors.surfaceContainerLow, modifier = Modifier.fillMaxWidth()) {
        Column(Modifier.padding(CardPadding), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(kindName(kind.kind), style = MaterialTheme.typography.titleMedium, modifier = Modifier.weight(1f))
                TextButton(onClick = onRemove, colors = ButtonDefaults.textButtonColors(contentColor = colors.error)) {
                    Text(stringResource(R.string.tile_remove))
                }
            }
            if (!placed) PackNote(R.string.pack_custom_tile_left_out)
            if (kind.forms.size > 1) {
                FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    kind.forms.forEach { form ->
                        FilterChip(
                            selected = form == tile.form,
                            onClick = { onChange(tile.copy(form = form)) },
                            label = { Text(FORM_NAMES[form]?.let { stringResource(it) } ?: form) },
                        )
                    }
                }
            }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                kind.sizes.forEach { size ->
                    SizeOption(size, selected = size == tile.size, enabled = size in fits) { onChange(tile.copy(size = size)) }
                }
            }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedButton(onClick = { onMove(-1) }, enabled = canMoveEarlier) { Text(stringResource(R.string.tile_move_earlier)) }
                OutlinedButton(onClick = { onMove(1) }, enabled = canMoveLater) { Text(stringResource(R.string.tile_move_later)) }
            }
        }
    }
}

@Composable
private fun SizeOption(size: String, selected: Boolean, enabled: Boolean, onClick: () -> Unit) {
    val colors = MaterialTheme.colorScheme
    val (cols, rows) = size.split('x').map { it.toInt() }
    val filled = if (selected) colors.primary else colors.onSurfaceVariant
    val empty = colors.outlineVariant
    Surface(
        onClick = onClick,
        enabled = enabled || selected,
        shape = MaterialTheme.shapes.small,
        color = if (selected) colors.secondaryContainer else colors.surfaceContainerHigh,
        border = if (selected) BorderStroke(2.dp, colors.primary) else null,
        modifier = Modifier.alpha(if (enabled || selected) 1f else DISABLED_ALPHA).semantics { contentDescription = size },
    ) {
        Canvas(Modifier.padding(8.dp).size(SizeGlyphCell * GLYPH_COLUMNS + SizeGlyphGap * (GLYPH_COLUMNS - 1), SizeGlyphCell * GLYPH_ROWS + SizeGlyphGap)) {
            val cell = SizeGlyphCell.toPx()
            val step = cell + SizeGlyphGap.toPx()
            val corner = CornerRadius(2.dp.toPx())
            for (r in 0 until GLYPH_ROWS) for (c in 0 until GLYPH_COLUMNS) {
                drawRoundRect(empty, Offset(c * step, r * step), Size(cell, cell), corner)
            }
            drawRoundRect(filled, Offset.Zero, Size(cols * step - SizeGlyphGap.toPx(), rows * step - SizeGlyphGap.toPx()), corner)
        }
    }
}

@Composable
private fun PackNote(@StringRes text: Int) {
    Text(stringResource(text), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.outline)
}
