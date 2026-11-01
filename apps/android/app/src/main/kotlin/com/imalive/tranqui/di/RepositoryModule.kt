package com.imalive.tranqui.di

import com.imalive.tranqui.feature.checkin.AuthRepository
import com.imalive.tranqui.feature.checkin.ContactsRepository
import com.imalive.tranqui.feature.checkin.NetworkAuthRepository
import com.imalive.tranqui.feature.checkin.NetworkContactsRepository
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

    @Binds
    abstract fun bindAuthRepository(impl: NetworkAuthRepository): AuthRepository

    @Binds
    abstract fun bindContactsRepository(impl: NetworkContactsRepository): ContactsRepository
}
