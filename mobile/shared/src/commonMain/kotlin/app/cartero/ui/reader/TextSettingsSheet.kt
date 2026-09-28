package app.cartero.ui.reader

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Slider
import androidx.compose.material3.SliderState
import androidx.compose.material3.Text
import androidx.compose.material3.rememberSliderState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import app.cartero.resources.Res
import app.cartero.resources.ic_format_size
import org.jetbrains.compose.resources.painterResource

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TextSettingsSheet(scale: Float, onScale: (Float) -> Unit, onDismiss: () -> Unit) {
    val commit by rememberUpdatedState(onScale)
    lateinit var slider: SliderState
    slider = rememberSliderState(
        value = scale,
        steps = 6,
        onValueChangeFinished = { commit(slider.value) },
        valueRange = MIN_SCALE..MAX_SCALE,
    )
    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(
            modifier = Modifier.padding(start = 24.dp, end = 24.dp, bottom = 24.dp),
            verticalArrangement = Arrangement.spacedBy(20.dp),
        ) {
            Text("Text size", style = MaterialTheme.typography.titleLarge)
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Icon(painterResource(Res.drawable.ic_format_size), contentDescription = null, modifier = Modifier.padding(4.dp))
                Slider(state = slider, modifier = Modifier.weight(1f))
            }
        }
    }
}

private const val MIN_SCALE = 0.85f
private const val MAX_SCALE = 1.4f
