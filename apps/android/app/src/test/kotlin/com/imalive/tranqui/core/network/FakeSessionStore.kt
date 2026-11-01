package com.imalive.tranqui.core.network

import com.imalive.tranqui.core.session.SessionStore
import com.imalive.tranqui.core.session.StoredSession
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow

// In-memory SessionStore for the MockWebServer tests: the same contract the
// DataStore implementation honors, without touching disk.
class FakeSessionStore(
    initial: StoredSession? = null,
    complete: Boolean = false,
) : SessionStore {
    private val state = MutableStateFlow(initial)
    private val completeState = MutableStateFlow(complete)
    var clearCount: Int = 0
        private set

    override val session: Flow<StoredSession?> = state.asStateFlow()

    override val onboardingComplete: Flow<Boolean> = completeState.asStateFlow()

    override suspend fun save(session: StoredSession) {
        state.value = session
    }

    override suspend fun saveUserPhone(phone: String) {
        state.value = state.value?.copy(userPhone = phone)
    }

    override suspend fun completeOnboarding() {
        completeState.value = true
    }

    override suspend fun clear() {
        state.value = null
        completeState.value = false // clear() wipes every key, the flag too
        clearCount++
    }

    override fun cachedSession(): StoredSession? = state.value
}
