package com.imalive.tranqui.core.network

import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import retrofit2.converter.kotlinx.serialization.asConverterFactory

/**
 * The single place where the HTTP stack is assembled, shared by Hilt
 * ([di.NetworkModule], base URL from BuildConfig) and by the unit tests
 * (base URL = MockWebServer). Keeping it factory-shaped keeps the base URL
 * injectable, per docs/stack/android.md.
 */
object ApiClientFactory {
    /** Contract JSON dialect: additive-safe decoding (plan issue #75). */
    fun json(): Json =
        Json {
            ignoreUnknownKeys = true
            explicitNulls = false
        }

    /**
     * Logging only for debug builds and at BASIC level (request/response
     * lines, never bodies): phone and OTP ride in bodies and must not land
     * in logcat. The Authorization header is redacted as defense in depth.
     */
    fun loggingInterceptor(debug: Boolean): HttpLoggingInterceptor =
        HttpLoggingInterceptor().apply {
            level = if (debug) HttpLoggingInterceptor.Level.BASIC else HttpLoggingInterceptor.Level.NONE
            redactHeader("Authorization")
        }

    fun okHttpClient(
        authInterceptor: AuthInterceptor,
        authenticator: TokenAuthenticator,
        logging: HttpLoggingInterceptor,
    ): OkHttpClient =
        OkHttpClient
            .Builder()
            .addInterceptor(authInterceptor)
            .authenticator(authenticator)
            .addInterceptor(logging)
            .build()

    fun retrofit(
        baseUrl: String,
        client: OkHttpClient,
        json: Json,
    ): Retrofit =
        Retrofit
            .Builder()
            .baseUrl(baseUrl)
            .client(client)
            .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
            .build()

    fun tranquiApi(retrofit: Retrofit): TranquiApi = retrofit.create(TranquiApi::class.java)
}
