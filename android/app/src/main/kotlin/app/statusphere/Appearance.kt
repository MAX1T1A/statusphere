package app.statusphere

import androidx.annotation.StringRes
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Checkbox
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.PrimaryTabRow
import androidx.compose.material3.Surface
import androidx.compose.material3.Tab
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import app.statusphere.mobile.Mobile
import kotlinx.coroutines.launch
import org.json.JSONArray
import org.json.JSONObject

data class PackChoice(val id: String, val preview: Card)

data class PackChoices(val active: String, val packs: List<PackChoice>)

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

fun List<CustomTile>.toJSON(): String =
    JSONArray(map { JSONObject().put("kind", it.kind).put("form", it.form).put("size", it.size) }).toString()

private enum class CardSurface(val wire: String, @StringRes val title: Int) {
    ROW(Mobile.RowSurface, R.string.appearance_row),
    DETAIL(Mobile.DetailSurface, R.string.appearance_detail),
}

private val PACK_NAMES = mapOf(
    Mobile.DefaultPack to R.string.pack_default,
    "cover" to R.string.pack_cover,
    "vinyl" to R.string.pack_vinyl,
    "minimal" to R.string.pack_minimal,
    "music" to R.string.pack_music,
    "dashboard" to R.string.pack_dashboard,
    "compact" to R.string.pack_compact,
)

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

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AppearanceSheet(onDismiss: () -> Unit) {
    val scope = rememberCoroutineScope()
    var surface by rememberSaveable { mutableStateOf(CardSurface.ROW) }
    var editing by rememberSaveable(surface) { mutableStateOf(false) }
    var choices by remember { mutableStateOf<PackChoices?>(null) }
    var error by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(surface, editing) {
        choices = null
        if (editing) return@LaunchedEffect
        PresenceService.packChoices(surface.wire)
            .onSuccess { choices = it }
            .onFailure { error = it.message ?: it.javaClass.simpleName }
    }

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
            if (editing) {
                CustomEditor(surface, onBack = { editing = false }, onError = { error = it })
            } else choices?.let { current ->
                current.packs.forEach { pack ->
                    PackOption(surface, pack, selected = pack.id == current.active) {
                        error = null
                        scope.launch {
                            PresenceService.setPack(surface.wire, pack.id)
                                .onSuccess { choices = current.copy(active = pack.id) }
                                .onFailure { error = it.message ?: it.javaClass.simpleName }
                        }
                    }
                }
                CustomOption(selected = current.active == Mobile.CustomPack) { editing = true }
            }
        }
    }
}

@Composable
private fun PackOption(surface: CardSurface, pack: PackChoice, selected: Boolean, onPick: () -> Unit) {
    val colors = MaterialTheme.colorScheme
    Surface(
        onClick = onPick,
        shape = CardShape,
        color = colors.surfaceContainerLow,
        border = if (selected) BorderStroke(2.dp, colors.primary) else null,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(Modifier.padding(CardPadding), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(PACK_NAMES[pack.id]?.let { stringResource(it) } ?: pack.id, style = MaterialTheme.typography.bodyLarge)
            SurfacePreview(surface, pack.preview)
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
private fun CustomOption(selected: Boolean, onPick: () -> Unit) {
    val colors = MaterialTheme.colorScheme
    Surface(
        onClick = onPick,
        shape = CardShape,
        color = colors.surfaceContainerLow,
        border = if (selected) BorderStroke(2.dp, colors.primary) else null,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(Modifier.padding(CardPadding), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(stringResource(R.string.pack_custom), style = MaterialTheme.typography.bodyLarge)
            PackNote(R.string.pack_custom_note)
        }
    }
}

@Composable
private fun CustomEditor(surface: CardSurface, onBack: () -> Unit, onError: (String) -> Unit) {
    var kinds by remember { mutableStateOf<List<TileKind>>(emptyList()) }
    var chosen by remember { mutableStateOf<List<CustomTile>?>(null) }
    var preview by remember { mutableStateOf<Card?>(null) }

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

    fun edit(kind: TileKind, tile: CustomTile?) {
        val current = chosen ?: return
        val next = kinds.mapNotNull { k -> if (k == kind) tile else current.find { it.kind == k.kind } }
        chosen = next
        PresenceService.saveCustomSoon(surface.wire, next)
    }

    TextButton(onClick = onBack) { Text(stringResource(R.string.pack_custom_back)) }
    val current = chosen ?: return
    if (current.isEmpty() && surface == CardSurface.DETAIL) PackNote(R.string.pack_custom_empty_detail)
    else preview?.let { SurfacePreview(surface, it) }
    kinds.forEach { kind ->
        val tile = current.find { it.kind == kind.kind }
        CustomTileRow(kind, tile) { edit(kind, it) }
    }
}

@Composable
private fun CustomTileRow(kind: TileKind, tile: CustomTile?, onChange: (CustomTile?) -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Checkbox(
                checked = tile != null,
                onCheckedChange = { on ->
                    val size = NEW_TILE_SIZE.takeIf { it in kind.sizes } ?: kind.sizes.first()
                    onChange(if (on) CustomTile(kind.kind, kind.forms.first(), size) else null)
                },
            )
            Text(KIND_NAMES[kind.kind]?.let { stringResource(it) } ?: kind.kind, style = MaterialTheme.typography.bodyLarge)
        }
        if (tile == null) return@Column
        val chips = Modifier.padding(start = 48.dp)
        if (kind.forms.size > 1) {
            FlowRow(chips, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                kind.forms.forEach { form ->
                    FilterChip(
                        selected = form == tile.form,
                        onClick = { onChange(tile.copy(form = form)) },
                        label = { Text(FORM_NAMES[form]?.let { stringResource(it) } ?: form) },
                    )
                }
            }
        }
        FlowRow(chips, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            kind.sizes.forEach { size ->
                FilterChip(selected = size == tile.size, onClick = { onChange(tile.copy(size = size)) }, label = { Text(size) })
            }
        }
    }
}

@Composable
private fun PackNote(@StringRes text: Int) {
    Text(stringResource(text), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.outline)
}
