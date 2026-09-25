package app.statusphere

import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
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
import androidx.compose.material3.Icon
import androidx.compose.material3.LocalContentColor
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.TextLayoutResult
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.em
import androidx.compose.ui.unit.max
import androidx.compose.ui.unit.min
import androidx.compose.ui.unit.sp
import kotlin.math.PI
import kotlin.math.cos
import kotlin.math.roundToInt
import kotlin.math.sin
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
private val TileIconSize = 14.dp
private val VideoIconSize = 28.dp

private const val DIAL_WAVE_AMPLITUDE_FRACTION = 0.012f
private val MinDialWaveAmplitude = 1.5.dp
private const val DIAL_WAVE_LENGTH_FRACTION = 0.12f
private val MinDialWaveLength = 12.dp
private const val DIAL_WAVE_PERIOD_MS = 2000

private val ValueSize = TextAutoSize.StepBased(minFontSize = 10.sp, maxFontSize = 40.sp)
private val SentenceSize = TextAutoSize.StepBased(minFontSize = 10.sp, maxFontSize = 28.sp)
private val SentenceLineHeight = 1.2.em

// Names come from the icon field emitted by cardlayout.go, Material Symbols Rounded ligatures.
private val TileIcons = mapOf(
    "planner_review" to R.drawable.ic_tile_planner_review,
    "memory" to R.drawable.ic_tile_memory,
    "storage" to R.drawable.ic_tile_storage,
    "deployed_code" to R.drawable.ic_tile_deployed_code,
    "terminal" to R.drawable.ic_tile_terminal,
    "desktop_windows" to R.drawable.ic_tile_desktop_windows,
    "code" to R.drawable.ic_tile_code,
    "mood" to R.drawable.ic_tile_mood,
    "location_on" to R.drawable.ic_tile_location_on,
    "album" to R.drawable.ic_tile_album,
    "library_music" to R.drawable.ic_tile_library_music,
    "speed" to R.drawable.ic_tile_speed,
    "schedule" to R.drawable.ic_tile_schedule,
    "web_asset" to R.drawable.ic_tile_web_asset,
    "apps" to R.drawable.ic_tile_apps,
    "inventory_2" to R.drawable.ic_tile_inventory_2,
)

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
                TileType.VIDEO -> VideoTile(tile, inset)
                TileType.GAME, TileType.PHOTO, TileType.PICTURE -> PictureTile(tile, inset)
                TileType.SCALAR, null -> when (tile.form) {
                    ScalarForm.RING -> RingTile(tile, inset)
                    ScalarForm.DIAL -> DialTile(tile, inset)
                    ScalarForm.BAR -> BarTile(tile, inset)
                    ScalarForm.NUMBER -> NumberTile(tile, inset)
                    ScalarForm.TEXT, ScalarForm.SUN -> TextTile(tile, inset)
                    ScalarForm.BIG, ScalarForm.MOON -> StickerTile(tile, inset)
                    ScalarForm.CLOCK -> ClockTile(tile, inset)
                    ScalarForm.WEATHER -> WeatherTile(tile, inset)
                    ScalarForm.WEATHER_LIVE -> WeatherLiveTile(tile, inset)
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
private fun TileLabel(
    text: String,
    modifier: Modifier = Modifier,
    textAlign: TextAlign? = null,
    onTextLayout: (TextLayoutResult) -> Unit = {},
) {
    Text(
        text,
        modifier,
        color = mutedColor(),
        style = MaterialTheme.typography.labelMedium,
        textAlign = textAlign,
        maxLines = 1,
        overflow = TextOverflow.Ellipsis,
        onTextLayout = onTextLayout,
    )
}

// Shows "label · note" only when it fits on one line, matching NotedLabel.qml.
@Composable
private fun NotedLabel(tile: Tile, modifier: Modifier = Modifier, textAlign: TextAlign? = null) {
    if (tile.note.isEmpty()) {
        TileLabel(tile.label, modifier, textAlign)
        return
    }
    var fits by remember(tile.label, tile.note) { mutableStateOf(true) }
    TileLabel(
        if (fits) "${tile.label} · ${tile.note}" else tile.label,
        modifier,
        textAlign,
        onTextLayout = { fits = !it.hasVisualOverflow },
    )
}

@Composable
private fun TileIcon(name: String) {
    val res = TileIcons[name] ?: return
    Icon(painterResource(res), contentDescription = null, tint = mutedColor(), modifier = Modifier.size(TileIconSize))
}

@Composable
private fun TextTile(tile: Tile, modifier: Modifier) {
    Column(modifier.fillMaxSize(), verticalArrangement = Arrangement.spacedBy(4.dp)) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(4.dp), verticalAlignment = Alignment.CenterVertically) {
            TileIcon(tile.icon)
            NotedLabel(tile, Modifier.weight(1f))
        }
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
        NotedLabel(tile)
        Box(Modifier.weight(1f).fillMaxWidth(), contentAlignment = Alignment.CenterStart) {
            Text(tile.percentText, autoSize = SentenceSize, maxLines = 1)
        }
        val content = LocalContentColor.current
        WavyProgress(tile.fraction, wavy = false, Modifier.fillMaxWidth(), content, content.copy(alpha = TRACK_ALPHA))
    }
}

@Composable
private fun RingTile(tile: Tile, modifier: Modifier) = RingLikeTile(tile, modifier, wavy = false)

@Composable
private fun DialTile(tile: Tile, modifier: Modifier) = RingLikeTile(tile, modifier, wavy = true)

@Composable
private fun RingLikeTile(tile: Tile, modifier: Modifier, wavy: Boolean) {
    val content = LocalContentColor.current
    val wavePhase = if (wavy) dialWavePhase() else null
    BoxWithConstraints(modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        val diameter = min(maxWidth, maxHeight)
        val stroke = max(MinRingStroke, diameter * RING_STROKE_FRACTION)
        val innerBox = (diameter - stroke * 2) * sqrt(0.5f)
        Canvas(Modifier.size(diameter)) {
            val width = stroke.toPx()
            val topLeft = Offset(width / 2, width / 2)
            val arc = Size(size.width - width, size.height - width)
            drawArc(content.copy(alpha = TRACK_ALPHA), 0f, 360f, false, topLeft, arc, style = Stroke(width))
            if (wavePhase == null) {
                drawArc(content, -90f, 360f * tile.fraction, false, topLeft, arc, style = Stroke(width, cap = StrokeCap.Round))
            } else {
                val amplitude = kotlin.math.max(MinDialWaveAmplitude.toPx(), size.minDimension * DIAL_WAVE_AMPLITUDE_FRACTION)
                val waveLength = kotlin.math.max(MinDialWaveLength.toPx(), size.minDimension * DIAL_WAVE_LENGTH_FRACTION)
                val radius = (size.minDimension - width) / 2
                val path = dialWavePath(center, radius, amplitude, waveLength, tile.fraction, wavePhase())
                drawPath(path, content, style = Stroke(width, cap = StrokeCap.Round))
            }
        }
        Column(Modifier.size(innerBox), Arrangement.Center, Alignment.CenterHorizontally) {
            Text(tile.percentText, autoSize = ValueSize, maxLines = 1, textAlign = TextAlign.Center)
            if (innerBox >= MinRingBoxForCaption) TileLabel(tile.label, textAlign = TextAlign.Center)
        }
    }
}

@Composable
private fun dialWavePhase(): () -> Float {
    val phase by rememberInfiniteTransition(label = "dial_wave").animateFloat(
        initialValue = 0f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(tween(DIAL_WAVE_PERIOD_MS, easing = LinearEasing)),
        label = "dial_wave_phase",
    )
    return { phase }
}

// Ported from WavyRing.qml's PathPolyline: an arc from -90deg whose radius is
// modulated by a travelling sine wave, tapered to the plain radius at both ends.
private fun dialWavePath(center: Offset, radius: Float, amplitude: Float, waveLength: Float, fraction: Float, phase: Float): Path {
    val degree = 360f * fraction.coerceIn(0f, 1f)
    val path = Path()
    if (degree <= 0f) return path
    val steps = kotlin.math.max(20, (degree * 1.5f).roundToInt())
    val waveFrequency = 2 * PI.toFloat() * radius / waveLength
    for (i in 0..steps) {
        val currentDeg = -90f + degree * i / steps
        var edgeFactor = 1f
        if (i < 4) edgeFactor = i / 4f
        if (steps - i < 4) edgeFactor = (steps - i) / 4f
        val waveOffset = amplitude * sin((currentDeg * waveFrequency + phase * 360f) * PI.toFloat() / 180f) * edgeFactor
        val r = radius + waveOffset
        val rad = currentDeg * PI.toFloat() / 180f
        val point = Offset(center.x + r * cos(rad), center.y + r * sin(rad))
        if (i == 0) path.moveTo(point.x, point.y) else path.lineTo(point.x, point.y)
    }
    return path
}

@Composable
private fun StickerTile(tile: Tile, modifier: Modifier) {
    Column(modifier.fillMaxSize(), verticalArrangement = Arrangement.spacedBy(2.dp), horizontalAlignment = Alignment.CenterHorizontally) {
        NotedLabel(tile, textAlign = TextAlign.Center)
        Box(Modifier.weight(1f).fillMaxWidth(), contentAlignment = Alignment.Center) {
            Text(tile.valueText, autoSize = ValueSize, maxLines = 3, textAlign = TextAlign.Center)
        }
    }
}

@Composable
private fun ClockTile(tile: Tile, modifier: Modifier) {
    Box(modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        Text(tile.valueText, autoSize = ValueSize, maxLines = 3, textAlign = TextAlign.Center)
    }
}

// "12° · Sunny · City" -> ("12°", "City"), matching TileWeather.qml's regex split.
private val WeatherTempPattern = Regex("""-?\d+°""")

private fun weatherFields(text: String): Pair<String, String> {
    val temp = WeatherTempPattern.find(text)?.value
    val rest = (temp?.let { text.replace(it, "") } ?: text).trim(' ', '·', ',', '-')
    return (temp ?: text) to rest.substringAfterLast('·').trim()
}

@Composable
private fun WeatherTile(tile: Tile, modifier: Modifier) {
    val (value, city) = remember(tile.value) { weatherFields(tile.valueText) }
    NumberLikeTile(value, city, modifier)
}

// wttr.in-derived compact string: temp;code;precipMM;windKmph;windDirDeg;isDay;moonIllum;
// moonPhase;sunriseMin;sunsetMin;nowMin;city - matches CardLayouts.js's weatherFieldsOf.
private val WeatherLivePattern = Regex("""^(-?\d+);\d+;[\d.]+;[\d.]+;[\d.]+;[01];\d+;[^;]*;\d+;\d+;\d+;(.*)$""")

@Composable
private fun WeatherLiveTile(tile: Tile, modifier: Modifier) {
    val fields = remember(tile.value) { WeatherLivePattern.find(tile.value) }
    val (value, city) = if (fields != null) "${fields.groupValues[1]}°" to fields.groupValues[2] else tile.valueText to ""
    NumberLikeTile(value, city, modifier)
}

@Composable
private fun NumberLikeTile(value: String, caption: String, modifier: Modifier) {
    Column(
        modifier.fillMaxSize(),
        verticalArrangement = Arrangement.spacedBy(2.dp, Alignment.CenterVertically),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        if (caption.isNotEmpty()) TileLabel(caption, textAlign = TextAlign.Center)
        Text(value, autoSize = ValueSize, maxLines = 1, textAlign = TextAlign.Center)
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
private fun VideoTile(tile: Tile, modifier: Modifier) {
    Row(modifier.fillMaxSize(), Arrangement.spacedBy(8.dp), Alignment.CenterVertically) {
        Icon(painterResource(R.drawable.ic_tile_smart_display), contentDescription = null, modifier = Modifier.size(VideoIconSize))
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Text(
                tile.title.ifEmpty { MISSING_VALUE },
                style = MaterialTheme.typography.bodyMedium,
                maxLines = 1,
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
            if (tile.percent != null) {
                val content = LocalContentColor.current
                WavyProgress(tile.fraction, wavy = false, Modifier.fillMaxWidth().padding(top = 4.dp), content, content.copy(alpha = TRACK_ALPHA))
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
