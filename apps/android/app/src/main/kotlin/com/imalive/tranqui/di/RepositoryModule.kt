package com.imalive.tranqui.di

import com.imalive.tranqui.feature.checkin.ProfileRepository
import com.imalive.tranqui.feature.checkin.StubProfileRepository
import dagger.Binds
import dagger.Module
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent

// The repository bindings of the app. When the network client lands with
// the auth flow (issue #75), the Retrofit implementation binds here.
@Module
@InstallIn(SingletonComponent::class)
abstract class RepositoryModule {
    @Binds
    abstract fun bindProfileRepository(impl: StubProfileRepository): ProfileRepository
}
