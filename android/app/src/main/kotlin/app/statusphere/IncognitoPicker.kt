package app.statusphere

import androidx.annotation.DrawableRes
import androidx.annotation.StringRes
import androidx.compose.animation.core.animateDpAsState
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.snap
import androidx.compose.animation.core.tween
import androidx.compose.animation.animateColorAsState
import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.awaitEachGesture
import androidx.compose.foundation.gestures.awaitFirstDown
import androidx.compose.foundation.gestures.waitForUpOrCancellation
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import app.statusphere.mobile.Mobile

// Same choices as PresenceIncognitoPicker.qml in the ii-widget-statusphere repo.
enum class IncognitoChoice(
    val on: Boolean,
    val minutes: Long,
    @StringRes val label: Int,
    @DrawableRes val icon: Int? = null,
    @StringRes val chip: Int? = null,
) {
    VISIBLE(on = false, Mobile.IncognitoUntilTurnedOff, R.string.incognito_visible, icon = R.drawable.ic_visibility),
    QUARTER_HOUR(on = true, 15, R.string.incognito_15m, chip = R.string.incognito_15m_chip),
    HOUR(on = true, 60, R.string.incognito_1h, chip = R.string.incognito_1h_chip),
    UNTIL_TURNED_OFF(on = true, Mobile.IncognitoUntilTurnedOff, R.string.incognito_until_turned_off, icon = R.drawable.ic_visibility_off),
}

private const val HOLD_MS = 420L
private const val CHIP_STAGGER_MS = 35
private val ChipSize = 34.dp
private val ChipRestDisc = ChipSize - 5.dp
private val ChipSpacing = 6.dp
private val HeaderSpacing = 12.dp
private val HoverSlack = 24.dp
private val RingStroke = 3.dp
private val RingGap = 5.dp

@Composable
fun IncognitoHeader(
    pickable: Boolean,
    avatarSize: Dp,
    onPick: (IncognitoChoice) -> Unit,
    avatar: @Composable () -> Unit,
    info: @Composable () -> Unit,
) {
    var holding by remember { mutableStateOf(false) }
    var open by remember { mutableStateOf(false) }
    var hovered by remember { mutableStateOf<IncognitoChoice?>(null) }
    val haptics = LocalHapticFeedback.current
    val ring = MaterialTheme.colorScheme.primary
    val progress by animateFloatAsState(
        if (holding) 1f else 0f,
        if (holding) tween(HOLD_MS.toInt(), easing = LinearEasing) else snap(),
        label = "hold",
    )

    val gesture = if (!pickable) Modifier else Modifier.pointerInput(onPick) {
        val avatarPx = avatarSize.toPx()
        val chipsStart = avatarPx + HeaderSpacing.toPx()
        val chipStep = (ChipSize + ChipSpacing).toPx()
        val slack = HoverSlack.toPx()
        fun choiceAt(at: Offset): IncognitoChoice? {
            val index = ((at.x - chipsStart) / chipStep).toInt()
            val inside = at.x >= chipsStart && at.y > -slack && at.y < size.height + slack
            return IncognitoChoice.entries.getOrNull(index)?.takeIf { inside }
        }
        awaitEachGesture {
            val down = awaitFirstDown(requireUnconsumed = false)
            if (down.position.x > avatarPx || down.position.y > avatarPx) return@awaitEachGesture
            holding = true
            var endedEarly = false
            withTimeoutOrNull(HOLD_MS) {
                waitForUpOrCancellation()
                endedEarly = true
            }
            holding = false
            if (endedEarly) return@awaitEachGesture

            open = true
            haptics.performHapticFeedback(HapticFeedbackType.LongPress)
            while (true) {
                val change = awaitPointerEvent().changes.firstOrNull { it.id == down.id } ?: break
                change.consume()
                hovered = choiceAt(change.position)
                if (!change.pressed) break
            }
            hovered?.let(onPick)
            open = false
            hovered = null
        }
    }

    Row(
        gesture,
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(HeaderSpacing),
    ) {
        Box(
            Modifier.drawBehind {
                if (progress == 0f) return@drawBehind
                val inset = -(RingGap + RingStroke / 2).toPx()
                drawArc(
                    ring,
                    startAngle = -90f,
                    sweepAngle = 360f * progress,
                    useCenter = false,
                    topLeft = Offset(inset, inset),
                    size = Size(size.width - 2 * inset, size.height - 2 * inset),
                    style = Stroke(RingStroke.toPx()),
                )
            },
        ) { avatar() }
        Box(Modifier.weight(1f), contentAlignment = Alignment.CenterStart) {
            Box(Modifier.alpha(if (open) 0f else 1f)) { info() }
            if (open) IncognitoChips(hovered)
        }
    }
}

@Composable
private fun IncognitoChips(hovered: IncognitoChoice?) {
    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(ChipSpacing)) {
        IncognitoChoice.entries.forEach { IncognitoChip(it, active = it == hovered) }
        Text(
            stringResource(hovered?.label ?: R.string.incognito_pick_hint),
            style = MaterialTheme.typography.bodySmall,
            color = if (hovered != null) MaterialTheme.colorScheme.onSurface else MaterialTheme.colorScheme.outline,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.padding(start = 4.dp),
        )
    }
}

@Composable
private fun IncognitoChip(choice: IncognitoChoice, active: Boolean) {
    val colors = MaterialTheme.colorScheme
    var shown by remember { mutableStateOf(false) }
    val disc by animateDpAsState(
        when {
            !shown -> 0.dp
            active -> ChipSize
            else -> ChipRestDisc
        },
        tween(delayMillis = if (active) 0 else choice.ordinal * CHIP_STAGGER_MS),
        label = "chip",
    )
    val fill by animateColorAsState(if (active) colors.secondaryContainer else colors.surfaceContainerHigh, label = "chipFill")
    val content = if (active) colors.onSecondaryContainer else colors.onSurface
    LaunchedEffect(Unit) { shown = true }

    Box(Modifier.size(ChipSize), contentAlignment = Alignment.Center) {
        Box(Modifier.size(disc).background(fill, CircleShape))
        choice.icon?.let { Icon(painterResource(it), contentDescription = null, tint = content, modifier = Modifier.size(18.dp)) }
        choice.chip?.let {
            Text(stringResource(it), style = MaterialTheme.typography.labelMedium, fontWeight = FontWeight.Medium, color = content)
        }
    }
}
