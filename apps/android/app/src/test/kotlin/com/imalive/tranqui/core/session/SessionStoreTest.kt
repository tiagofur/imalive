package com.imalive.tranqui.core.session

import androidx.datastore.preferences.core.PreferenceDataStoreFactory
import androidx.test.ext.junit.runners.AndroidJUnit4
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancelAndJoin
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.job
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.withContext
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TemporaryFolder
import org.junit.runner.RunWith
import org.robolectric.annotation.Config
import java.io.File

// Slice 1 of issue #75: tokens + phone round-trip in DataStore Preferences
// (docs/stack/android.md: DataStore for the session, cero datos de negocio).
@RunWith(AndroidJUnit4::class)
@Config(sdk = [35])
class SessionStoreTest {
    @get:Rule
    val tmpFolder = TemporaryFolder()

    private val openStores = mutableListOf<TestStore>()

    private val session =
        StoredSession(
            accessToken = "access-token-1",
            refreshToken = "refresh-token-1",
            expiresAtEpochSeconds = 1_793_500_800L,
            userPhone = "+5215512345678",
        )

    // One scope per store: DataStore allows a single active instance per
    // file, so tests close a store before another one opens the same file.
    private inner class TestStore(
        val folder: File,
    ) {
        val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())

        val store =
            DataStoreSessionStore(
                PreferenceDataStoreFactory.create(scope = scope) {
                    File(folder, "tranqui_session.preferences_pb")
                },
            )

        suspend fun close() {
            scope.coroutineContext.job.cancelAndJoin()
        }
    }

    private suspend fun newStore(folder: File = tmpFolder.newFolder("store-${openStores.size}")): TestStore {
        val testStore = TestStore(folder)
        openStores += testStore
        return testStore
    }

    @After
    fun tearDown(): Unit =
        runBlocking {
            withContext(Dispatchers.IO) {
                openStores.forEach { it.close() }
            }
            openStores.clear()
        }

    @Test
    fun `save then read returns the same session`() =
        runBlocking {
            val testStore = newStore()

            testStore.store.save(session)

            assertEquals(session, testStore.store.session.first())
            assertEquals(session, testStore.store.cachedSession())
        }

    @Test
    fun `a fresh store reads a saved session back from disk`() =
        runBlocking {
            val writer = newStore()
            writer.store.save(session)
            writer.close() // releases the file for the next instance

            val reader = newStore(writer.folder)

            assertEquals(session, reader.store.cachedSession())
        }

    @Test
    fun `saveUserPhone updates only the phone`() =
        runBlocking {
            val testStore = newStore()
            testStore.store.save(session.copy(userPhone = null))

            testStore.store.saveUserPhone("+5215512345678")

            val stored = testStore.store.cachedSession()
            assertEquals("access-token-1", stored?.accessToken)
            assertEquals("refresh-token-1", stored?.refreshToken)
            assertEquals("+5215512345678", stored?.userPhone)
        }

    @Test
    fun `clear wipes the session`() =
        runBlocking {
            val testStore = newStore()
            testStore.store.save(session)

            testStore.store.clear()

            assertNull(testStore.store.session.first())
            assertNull(testStore.store.cachedSession())
        }

    @Test
    fun `completeOnboarding round-trips and clear resets the flag`() =
        runBlocking {
            val testStore = newStore()

            assertEquals(false, testStore.store.onboardingComplete.first())

            testStore.store.completeOnboarding()
            assertEquals(true, testStore.store.onboardingComplete.first())

            testStore.store.clear()
            assertEquals(false, testStore.store.onboardingComplete.first())
        }

    @Test
    fun `an empty store reads as no session`() =
        runBlocking {
            val testStore = newStore()

            assertNull(testStore.store.cachedSession())
            assertNull(testStore.store.session.first())
        }
}
