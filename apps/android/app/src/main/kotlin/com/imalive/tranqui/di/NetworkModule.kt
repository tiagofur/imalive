package com.imalive.tranqui.di

import com.imalive.tranqui.BuildConfig
import com.imalive.tranqui.core.network.ApiClientFactory
import com.imalive.tranqui.core.network.AuthInterceptor
import com.imalive.tranqui.core.network.TokenAuthenticator
import com.imalive.tranqui.core.network.TranquiApi
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import kotlinx.serialization.json.Json
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import okhttp3.logging.HttpLoggingInterceptor.Level
import retrofit2.Retrofit
import javax.inject.Singleton

/**
 * The real network graph (issue #75 slice 1): base URL from BuildConfig,
 * OkHttp + Retrofit + kotlinx.serialization, with the bearer interceptor and
 * the single-flight refresh authenticator wired in.
 */
@Module
@InstallIn(SingletonComponent::class)
object NetworkModule {
    @Provides
    @Singleton
    fun provideJson(): Json = ApiClientFactory.json()

    @Provides
    @Singleton
    fun provideLoggingInterceptor(): HttpLoggingInterceptor {
        // Body of two statements on purpose: a single-return block makes
        // ktlint demand an expression body whose joined line then trips
        // detekt's MaxLineLength (issue #75 slice 1 review).
        val level = if (BuildConfig.DEBUG) Level.BODY else Level.NONE
        return HttpLoggingInterceptor().apply { this.level = level }
    }

    @Provides
    @Singleton
    fun provideOkHttpClient(
        authInterceptor: AuthInterceptor,
        authenticator: TokenAuthenticator,
        loggingInterceptor: HttpLoggingInterceptor,
    ): OkHttpClient = ApiClientFactory.okHttpClient(authInterceptor, authenticator, loggingInterceptor)

    @Provides
    @Singleton
    fun provideRetrofit(
        client: OkHttpClient,
        json: Json,
    ): Retrofit = ApiClientFactory.retrofit(BuildConfig.API_BASE_URL, client, json)

    @Provides
    @Singleton
    fun provideTranquiApi(retrofit: Retrofit): TranquiApi = ApiClientFactory.tranquiApi(retrofit)
}
