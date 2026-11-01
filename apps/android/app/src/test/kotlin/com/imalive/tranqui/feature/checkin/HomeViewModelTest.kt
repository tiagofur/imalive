package com.imalive.tranqui.feature.checkin

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import java.time.Instant

// Criterion of issue #72 (the home loads the streak through the repository
// seam) adapted in #75 slice 2 to the real Profile shape: the label follows
// the profile slot and the pause (R-11) is what freezes the streak. The
// fake repos stand in for the network repo, whose wire behavior is covered
// by NetworkRepositoriesTest. setMain keeps viewModelScope's init launch
// eager and deterministic on the JVM.
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

    private fun profile(
        streakDays: Int,
        checkinTime: String = "21:00",
        pauseUntil: Instant? = null,
    ): Profile = Profile(streakDays = streakDays, checkinTime = checkinTime, pauseUntil = pauseUntil)

    @Test
    fun `loads the streak from the repository`() =
        runTest {
            val viewModel = HomeViewModel(FixedProfileRepo(profile(streakDays = 12)))

            assertEquals(12, viewModel.uiState.value.streakDays)
            assertFalse(viewModel.uiState.value.streakFrozen)
        }

    @Test
    fun `the next-checkin label follows the profile slot`() =
        runTest {
            val viewModel = HomeViewModel(FixedProfileRepo(profile(streakDays = 3, checkinTime = "08:30")))

            assertEquals("hoy 08:30", viewModel.uiState.value.nextCheckinLabel)
        }

    @Test
    fun `marks the streak frozen while the pause is live`() =
        runTest {
            val livePause = Instant.now().plusSeconds(3600)
            val viewModel = HomeViewModel(FixedProfileRepo(profile(streakDays = 7, pauseUntil = livePause)))

            assertEquals(7, viewModel.uiState.value.streakDays)
            assertTrue(viewModel.uiState.value.streakFrozen)
        }

    @Test
    fun `an expired pause does not freeze the streak`() =
        runTest {
            val pastPause = Instant.now().minusSeconds(60)
            val viewModel = HomeViewModel(FixedProfileRepo(profile(streakDays = 7, pauseUntil = pastPause)))

            assertFalse(viewModel.uiState.value.streakFrozen)
        }

    @Test
    fun `a failed profile read keeps the home intact with the defaults`() =
        runTest {
            val viewModel = HomeViewModel(ExplodingRepo())

            assertEquals(0, viewModel.uiState.value.streakDays)
            assertFalse(viewModel.uiState.value.streakFrozen)
            assertEquals("hoy 21:00", viewModel.uiState.value.nextCheckinLabel)
        }
}
