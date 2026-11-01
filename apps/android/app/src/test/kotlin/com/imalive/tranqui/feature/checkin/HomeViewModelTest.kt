package com.imalive.tranqui.feature.checkin

import com.imalive.tranqui.core.network.FakeSessionStore
import com.imalive.tranqui.core.network.dto.ProfilePatch
import com.imalive.tranqui.core.session.StoredSession
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

// Criteria of issues #72 and #75 slice 2: the session gate decides what the
// home shows, and the streak loads through the repository seam only with a
// session. The fakes stand in for the network repo and the DataStore — the
// wire behavior is covered by NetworkRepositoriesTest and SessionStoreTest.
// setMain keeps viewModelScope's init launch eager and deterministic on the
// JVM.
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

        override suspend fun update(patch: ProfilePatch) { }
    }

    private class ExplodingRepo : ProfileRepository {
        override suspend fun profile(): Profile = throw java.io.IOException("network down")

        override suspend fun update(patch: ProfilePatch): Unit = throw java.io.IOException("network down")
    }

    private class CountingRepo(
        private val profile: Profile,
    ) : ProfileRepository {
        var calls: Int = 0
            private set

        override suspend fun profile(): Profile {
            calls++
            return profile
        }

        override suspend fun update(patch: ProfilePatch) {
            calls++
        }
    }

    private val session =
        StoredSession(
            accessToken = "access-token-1",
            refreshToken = "refresh-token-1",
            expiresAtEpochSeconds = 0L,
        )

    private fun profile(
        streakDays: Int,
        checkinTime: String = "21:00",
        pauseUntil: Instant? = null,
    ): Profile = Profile(streakDays = streakDays, checkinTime = checkinTime, pauseUntil = pauseUntil)

    // --- session gate (issue #75 slices 2 and 4) ---

    @Test
    fun `without a session the gate asks for onboarding and never reads the profile`() =
        runTest {
            val viewModel = HomeViewModel(ExplodingRepo(), FakeSessionStore(null))

            assertEquals(HomeGate.NEEDS_ONBOARDING, viewModel.uiState.value.gate)
            assertEquals(0, viewModel.uiState.value.streakDays)
        }

    @Test
    fun `with a session and a finished onboarding the home is ready and loads the real streak`() =
        runTest {
            val viewModel =
                HomeViewModel(
                    FixedProfileRepo(profile(streakDays = 12)),
                    FakeSessionStore(session, complete = true),
                )

            assertEquals(HomeGate.READY, viewModel.uiState.value.gate)
            assertEquals(12, viewModel.uiState.value.streakDays)
        }

    @Test
    fun `a session alone does not land home until the onboarding completes`() =
        runTest {
            val sessions = FakeSessionStore(session)
            val viewModel = HomeViewModel(ExplodingRepo(), sessions)

            assertEquals(HomeGate.NEEDS_ONBOARDING, viewModel.uiState.value.gate)
        }

    @Test
    fun `completing the onboarding lands the user on home`() =
        runTest {
            val sessions = FakeSessionStore(null)
            val viewModel = HomeViewModel(FixedProfileRepo(profile(streakDays = 5)), sessions)

            assertEquals(HomeGate.NEEDS_ONBOARDING, viewModel.uiState.value.gate)

            sessions.save(session)
            assertEquals(HomeGate.NEEDS_ONBOARDING, viewModel.uiState.value.gate)

            sessions.completeOnboarding()

            assertEquals(HomeGate.READY, viewModel.uiState.value.gate)
            assertEquals(5, viewModel.uiState.value.streakDays)
        }

    @Test
    fun `a session wiped by the authenticator walks the user back to onboarding`() =
        runTest {
            val sessions = FakeSessionStore(session, complete = true)
            val viewModel = HomeViewModel(FixedProfileRepo(profile(streakDays = 12)), sessions)

            assertEquals(HomeGate.READY, viewModel.uiState.value.gate)

            sessions.clear()

            assertEquals(HomeGate.NEEDS_ONBOARDING, viewModel.uiState.value.gate)
            // The previous user's data does not leak into the entry point.
            assertEquals(0, viewModel.uiState.value.streakDays)
        }

    @Test
    fun `a token rotation does not reload the streak`() =
        runTest {
            val repo = CountingRepo(profile(streakDays = 12))
            val sessions = FakeSessionStore(session, complete = true)
            val viewModel = HomeViewModel(repo, sessions)

            assertEquals(HomeGate.READY, viewModel.uiState.value.gate)

            sessions.save(session.copy(accessToken = "access-token-2"))

            assertEquals(HomeGate.READY, viewModel.uiState.value.gate)
            assertEquals(1, repo.calls)
        }

    // --- streak (issue #72, real shape of #75 slice 2) ---

    @Test
    fun `loads the streak from the repository`() =
        runTest {
            val viewModel =
                HomeViewModel(FixedProfileRepo(profile(streakDays = 12)), FakeSessionStore(session, complete = true))

            assertEquals(12, viewModel.uiState.value.streakDays)
            assertFalse(viewModel.uiState.value.streakFrozen)
        }

    @Test
    fun `the next-checkin label follows the profile slot`() =
        runTest {
            val viewModel =
                HomeViewModel(
                    FixedProfileRepo(profile(streakDays = 3, checkinTime = "08:30")),
                    FakeSessionStore(session, complete = true),
                )

            assertEquals("hoy 08:30", viewModel.uiState.value.nextCheckinLabel)
        }

    @Test
    fun `marks the streak frozen while the pause is live`() =
        runTest {
            val livePause = Instant.now().plusSeconds(3600)
            val viewModel =
                HomeViewModel(
                    FixedProfileRepo(profile(streakDays = 7, pauseUntil = livePause)),
                    FakeSessionStore(session, complete = true),
                )

            assertEquals(7, viewModel.uiState.value.streakDays)
            assertTrue(viewModel.uiState.value.streakFrozen)
        }

    @Test
    fun `an expired pause does not freeze the streak`() =
        runTest {
            val pastPause = Instant.now().minusSeconds(60)
            val viewModel =
                HomeViewModel(
                    FixedProfileRepo(profile(streakDays = 7, pauseUntil = pastPause)),
                    FakeSessionStore(session, complete = true),
                )

            assertFalse(viewModel.uiState.value.streakFrozen)
        }

    @Test
    fun `a failed profile read keeps the home intact with the defaults`() =
        runTest {
            val viewModel = HomeViewModel(ExplodingRepo(), FakeSessionStore(session, complete = true))

            assertEquals(HomeGate.READY, viewModel.uiState.value.gate)
            assertEquals(0, viewModel.uiState.value.streakDays)
            assertFalse(viewModel.uiState.value.streakFrozen)
            assertEquals("hoy 21:00", viewModel.uiState.value.nextCheckinLabel)
        }
}
