package com.imalive.tranqui.feature.checkin

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

// Criterion of issue #72: the ViewModel loads the streak through the
// repository seam — the fake repos stand in for the network client that
// lands with the auth flow (#75). setMain keeps viewModelScope's init
// launch eager and deterministic on the JVM.
class HomeViewModelTest {
    private val mainDispatcher = UnconfinedTestDispatcher()

    @Before
    fun setUp() {
        Dispatchers.setMain(mainDispatcher)
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    private class FixedProfileRepo(
        private val profile: Profile,
    ) : ProfileRepository {
        override suspend fun profile(): Profile = profile
    }

    private class ExplodingRepo : ProfileRepository {
        override suspend fun profile(): Profile = throw java.io.IOException("network down")
    }

    @Test
    fun `loads the streak from the repository`() =
        runTest {
            val viewModel = HomeViewModel(FixedProfileRepo(Profile(streakDays = 12, streakFrozen = false)))

            assertEquals(12, viewModel.uiState.value.streakDays)
            assertEquals(false, viewModel.uiState.value.streakFrozen)
        }

    @Test
    fun `marks the streak frozen while the pause is live`() =
        runTest {
            val viewModel = HomeViewModel(FixedProfileRepo(Profile(streakDays = 7, streakFrozen = true)))

            assertEquals(7, viewModel.uiState.value.streakDays)
            assertTrue(viewModel.uiState.value.streakFrozen)
        }

    @Test
    fun `a failed profile read keeps the home intact with the defaults`() =
        runTest {
            val viewModel = HomeViewModel(ExplodingRepo())

            assertEquals(0, viewModel.uiState.value.streakDays)
            assertEquals(false, viewModel.uiState.value.streakFrozen)
            assertEquals("hoy 21:00", viewModel.uiState.value.nextCheckinLabel)
        }
}
