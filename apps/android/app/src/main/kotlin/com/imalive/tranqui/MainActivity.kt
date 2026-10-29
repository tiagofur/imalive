package com.imalive.tranqui

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import dagger.hilt.android.AndroidEntryPoint

@AndroidEntryPoint
class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            MaterialTheme {
                // Placeholder shell: the home screen ("Estoy OK") lands with the
                // A2 PR of the F0 scaffolding (issue #3).
                Text(text = getString(R.string.app_name))
            }
        }
    }
}
