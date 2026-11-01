package com.imalive.tranqui.di

import com.imalive.tranqui.feature.checkin.NetworkProfileRepository
import com.imalive.tranqui.feature.checkin.ProfileRepository
import dagger.Binds
import dagger.Module
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent

// The repository bindings of the app: the real network implementations over
// the contract client (issue #75 slice 2; the F0 stub is gone).
@Module
@InstallIn(SingletonComponent::class)
abstract class RepositoryModule {
    @Binds
    abstract fun bindProfileRepository(impl: NetworkProfileRepository): ProfileRepository
}
