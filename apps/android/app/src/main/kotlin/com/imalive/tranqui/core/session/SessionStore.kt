package com.imalive.tranqui.core.session

import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.longPreferencesKey
import androidx.datastore.preferences.core.stringPreferencesKey
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.runBlocking
import javax.inject.Inject
import javax.inject.Singleton

/**
 * The persisted session: the JWT pair plus the phone that verified it
 * (contract v0.5.0). Tokens live only in the app-private DataStore and are
 * never logged (docs/stack/android.md, privacy: minimum data).
 */
data class StoredSession(
    val accessToken: String,
    val refreshToken: String,
    // Absolute expiry of the access token (epoch seconds).
    val expiresAtEpochSeconds: Long,
    // The phone that owns the session (E.164); attached after the OTP verify.
    val userPhone: String? = null,
)

/**
 * Session persistence seam. `save/clear/session` are the coroutine-facing
 * API; [cachedSession] is the synchronous read the OkHttp interceptor and
 * authenticator need (warmed once per process — never a blocking read per
 * request, docs/stack/android.md).
 */
interface SessionStore {
    val session: Flow<StoredSession?>

    suspend fun save(session: StoredSession)

    /** Attaches (or replaces) the user phone without touching the tokens. */
    suspend fun saveUserPhone(phone: String)

    /** Wipes the session: used on logout and on a rejected refresh. */
    suspend fun clear()

    /** Synchronous snapshot for OkHttp's interceptors (no per-request blocking). */
    fun cachedSession(): StoredSession?
}

@Singleton
class DataStoreSessionStore
    @Inject
    constructor(
        private val dataStore: DataStore<Preferences>,
    ) : SessionStore {
        // In-memory mirror kept in sync on every write; the first read warms
        // it from disk once (synchronized) so interceptors never hit DataStore
        // on the hot path.
        @Volatile
        private var cache: StoredSession? = null

        @Volatile
        private var cacheLoaded = false

        override val session: Flow<StoredSession?> = dataStore.data.map(::fromPreferences)

        override suspend fun save(session: StoredSession) {
            dataStore.edit { prefs ->
                prefs[ACCESS_TOKEN] = session.accessToken
                prefs[REFRESH_TOKEN] = session.refreshToken
                prefs[EXPIRES_AT] = session.expiresAtEpochSeconds
                session.userPhone?.let { prefs[USER_PHONE] = it }
            }
            cache = session
            cacheLoaded = true
        }

        override suspend fun saveUserPhone(phone: String) {
            dataStore.edit { it[USER_PHONE] = phone }
            val current = cachedSession()
            if (current != null) cache = current.copy(userPhone = phone)
        }

        override suspend fun clear() {
            dataStore.edit { it.clear() }
            cache = null
            cacheLoaded = true
        }

        override fun cachedSession(): StoredSession? {
            if (!cacheLoaded) {
                synchronized(this) {
                    if (!cacheLoaded) {
                        cache = runBlocking { session.first() }
                        cacheLoaded = true
                    }
                }
            }
            return cache
        }

        private fun fromPreferences(prefs: Preferences): StoredSession? {
            val access = prefs[ACCESS_TOKEN]
            val refresh = prefs[REFRESH_TOKEN]
            // A session exists only with both tokens present.
            return when {
                access == null || refresh == null -> null
                else ->
                    StoredSession(
                        accessToken = access,
                        refreshToken = refresh,
                        expiresAtEpochSeconds = prefs[EXPIRES_AT] ?: 0L,
                        userPhone = prefs[USER_PHONE],
                    )
            }
        }

        private companion object {
            val ACCESS_TOKEN = stringPreferencesKey("access_token")
            val REFRESH_TOKEN = stringPreferencesKey("refresh_token")
            val EXPIRES_AT = longPreferencesKey("expires_at_epoch_seconds")
            val USER_PHONE = stringPreferencesKey("user_phone")
        }
    }
