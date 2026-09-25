package app.statusphere

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.text.TextAutoSize
import androidx.compose.material3.LocalContentColor
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.em
import androidx.compose.ui.unit.max
import androidx.compose.ui.unit.min
import androidx.compose.ui.unit.sp
import kotlin.math.roundToInt
import kotlin.math.sqrt

private const val GRID_COLUMNS = 4
private val TileGap = 8.dp
private val MinInset = 4.dp
private const val INSET_FRACTION = 0.1f
private const val DIMMED_ALPHA = 0.45f
private const val MUTED_ALPHA = 0.65f
private const val TRACK_ALPHA = 0.25f
private const val COVER_ART_FRACTION = 0.4f
private const val RING_STROKE_FRACTION = 0.08f
private val MinRingStroke = 3.dp
private val MinRingBoxForCaption = 56.dp
private const val MISSING_VALUE = "-"

private val ValueSize = TextAutoSize.StepBased(minFontSize = 10.sp, maxFontSize = 40.sp)
private val SentenceSize = TextAutoSize.StepBased(minFontSize = 10.sp, maxFontSize = 28.sp)
private val SentenceLineHeight = 1.2.em

@Composable
fun TileGrid(tiles: List<Tile>, modifier: Modifier = Modifier) {
    BoxWithConstraints(modifier.fillMaxWidth()) {
        val cell = (maxWidth - TileGap * (GRID_COLUMNS - 1)) / GRID_COLUMNS
        Box(Modifier.fillMaxWidth().height(span(tiles.maxOfOrNull { it.row + it.rows } ?: 0, cell))) {
            tiles.forEach {
                TileSurface(
                    it,
                    Modifier
                        .offset(x = (cell + TileGap) * it.col, y = (cell + TileGap) * it.row)
                        .size(span(it.cols, cell), span(it.rows, cell)),
                )
            }
        }
    }
}

private fun span(cells: Int, cell: Dp): Dp = if (cells > 0) cell * cells + TileGap * (cells - 1) else 0.dp

@Composable
private fun TileSurface(tile: Tile, modifier: Modifier) {
    val (tint, content) = tile.color.colors()
    BoxWithConstraints(
        modifier
            .alpha(if (tile.dimmed) DIMMED_ALPHA else 1f)
            .clip(CardShape)
            .background(tint),
    ) {
        val inset = Modifier.padding(max(MinInset, min(maxWidth, maxHeight) * INSET_FRACTION))
        CompositionLocalProvider(LocalContentColor provides content) {
            when (tile.type) {
                TileType.MUSIC -> CoverTile(tile, inset)
                TileType.GAME, TileType.PHOTO, TileType.PICTURE -> PictureTile(tile, inset)
                TileType.SCALAR, null -> when (tile.form) {
                    ScalarForm.RING -> RingTile(tile, inset)
                    ScalarForm.BAR -> BarTile(tile, inset)
                    ScalarForm.NUMBER -> NumberTile(tile, inset)
                    ScalarForm.TEXT -> TextTile(tile, inset)
                }
            }
        }
    }
}

@Composable
private fun TileColor?.colors(): Pair<Color, Color> {
    val scheme = MaterialTheme.colorScheme
    return when (this) {
        TileColor.PRIMARY -> scheme.primary to scheme.onPrimary
        TileColor.SECONDARY -> scheme.secondary to scheme.onSecondary
        TileColor.TERTIARY -> scheme.tertiary to scheme.onTertiary
        TileColor.ERROR -> scheme.error to scheme.onError
        TileColor.PRIMARY_CONTAINER -> scheme.primaryContainer to scheme.onPrimaryContainer
        TileColor.SECONDARY_CONTAINER -> scheme.secondaryContainer to scheme.onSecondaryContainer
        TileColor.TERTIARY_CONTAINER -> scheme.tertiaryContainer to scheme.onTertiaryContainer
        TileColor.ERROR_CONTAINER -> scheme.errorContainer to scheme.onErrorContainer
        null -> scheme.surfaceContainer to scheme.onSurface
    }
}

private val Tile.valueText: String get() = value.ifEmpty { MISSING_VALUE }

private val Tile.percentText: String get() = percent?.let { "${it.roundToInt()}%" } ?: MISSING_VALUE

private val Tile.fraction: Float get() = (percent ?: 0f).coerceIn(0f, 100f) / 100

@Composable
private fun mutedColor(): Color = LocalContentColor.current.copy(alpha = MUTED_ALPHA)

@Composable
private fun TileLabel(text: String, modifier: Modifier = Modifier, textAlign: TextAlign? = null) {
    Text(
        text,
        modifier,
        color = mutedColor(),
        style = MaterialTheme.typography.labelMedium,
        textAlign = textAlign,
        maxLines = 1,
        overflow = TextOverflow.Ellipsis,
    )
}

@Composable
private fun TextTile(tile: Tile, modifier: Modifier) {
    Column(modifier.fillMaxSize(), verticalArrangement = Arrangement.spacedBy(4.dp)) {
        TileLabel(tile.label)
        Box(Modifier.weight(1f).fillMaxWidth(), contentAlignment = Alignment.CenterStart) {
            Text(tile.valueText, autoSize = SentenceSize, lineHeight = SentenceLineHeight, overflow = TextOverflow.Ellipsis)
        }
    }
}

@Composable
private fun NumberTile(tile: Tile, modifier: Modifier) {
    Column(
        modifier.fillMaxSize(),
        verticalArrangement = Arrangement.spacedBy(2.dp, Alignment.CenterVertically),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        TileLabel(tile.label, textAlign = TextAlign.Center)
        Text(tile.valueText, autoSize = ValueSize, maxLines = 1, textAlign = TextAlign.Center)
    }
}

@Composable
private fun BarTile(tile: Tile, modifier: Modifier) {
    Column(modifier.fillMaxSize(), verticalArrangement = Arrangement.spacedBy(4.dp)) {
        TileLabel(tile.label)
        Box(Modifier.weight(1f).fillMaxWidth(), contentAlignment = Alignment.CenterStart) {
            Text(tile.percentText, autoSize = SentenceSize, maxLines = 1)
        }
        val content = LocalContentColor.current
        WavyProgress(tile.fraction, wavy = false, Modifier.fillMaxWidth(), content, content.copy(alpha = TRACK_ALPHA))
    }
}

@Composable
private fun RingTile(tile: Tile, modifier: Modifier) {
    val content = LocalContentColor.current
    BoxWithConstraints(modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        val diameter = min(maxWidth, maxHeight)
        val stroke = max(MinRingStroke, diameter * RING_STROKE_FRACTION)
        val innerBox = (diameter - stroke * 2) * sqrt(0.5f)
        Canvas(Modifier.size(diameter)) {
            val width = stroke.toPx()
            val topLeft = Offset(width / 2, width / 2)
            val arc = Size(size.width - width, size.height - width)
            drawArc(content.copy(alpha = TRACK_ALPHA), 0f, 360f, false, topLeft, arc, style = Stroke(width))
            drawArc(content, -90f, 360f * tile.fraction, false, topLeft, arc, style = Stroke(width, cap = StrokeCap.Round))
        }
        Column(Modifier.size(innerBox), Arrangement.Center, Alignment.CenterHorizontally) {
            Text(tile.percentText, autoSize = ValueSize, maxLines = 1, textAlign = TextAlign.Center)
            if (innerBox >= MinRingBoxForCaption) TileLabel(tile.label, textAlign = TextAlign.Center)
        }
    }
}

@Composable
private fun CoverTile(tile: Tile, modifier: Modifier) {
    BoxWithConstraints(modifier.fillMaxSize()) {
        val side = min(maxHeight, maxWidth * COVER_ART_FRACTION)
        Row(Modifier.fillMaxSize(), Arrangement.spacedBy(8.dp), Alignment.CenterVertically) {
            rememberArt(tile.imageUrl)?.let {
                Image(
                    it.asImageBitmap(),
                    contentDescription = null,
                    contentScale = ContentScale.Crop,
                    modifier = Modifier.size(side).clip(ArtShape),
                )
            }
            Column(Modifier.weight(1f)) {
                Text(
                    tile.title.ifEmpty { MISSING_VALUE },
                    style = MaterialTheme.typography.bodyMedium,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                if (tile.subtitle.isNotEmpty()) {
                    Text(
                        tile.subtitle,
                        color = mutedColor(),
                        style = MaterialTheme.typography.bodySmall,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
        }
    }
}

@Composable
private fun PictureTile(tile: Tile, modifier: Modifier) {
    val art = rememberArt(tile.imageUrl)
    if (art != null) {
        Image(
            art.asImageBitmap(),
            contentDescription = tile.title.ifEmpty { null },
            contentScale = ContentScale.Crop,
            modifier = Modifier.fillMaxSize(),
        )
    } else if (tile.title.isNotEmpty()) {
        Text(tile.title, modifier, style = MaterialTheme.typography.bodyMedium, overflow = TextOverflow.Ellipsis)
    }
}
