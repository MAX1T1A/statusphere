package app.statusphere

import androidx.annotation.StringRes
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.PrimaryTabRow
import androidx.compose.material3.Surface
import androidx.compose.material3.Tab
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import app.statusphere.mobile.Mobile
import kotlinx.coroutines.launch

data class PackChoice(val id: String, val preview: Card)

data class PackChoices(val active: String, val packs: List<PackChoice>)

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

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AppearanceSheet(onDismiss: () -> Unit) {
    val scope = rememberCoroutineScope()
    var surface by rememberSaveable { mutableStateOf(CardSurface.ROW) }
    var choices by remember { mutableStateOf<PackChoices?>(null) }
    var error by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(surface) {
        choices = null
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
            choices?.let { current ->
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
            val tiles = if (surface == CardSurface.ROW) pack.preview.row else pack.preview.detail
            when {
                tiles == null -> PackNote(R.string.pack_default_row)
                tiles.isEmpty() && surface == CardSurface.ROW -> PackNote(R.string.pack_header_only)
                tiles.isEmpty() -> PackNote(R.string.pack_nothing_yet)
                else -> TileGrid(tiles)
            }
        }
    }
}

@Composable
private fun PackNote(@StringRes text: Int) {
    Text(stringResource(text), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.outline)
}
