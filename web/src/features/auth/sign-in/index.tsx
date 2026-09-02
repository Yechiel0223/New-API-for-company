/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useSearch } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { AuthLayout } from '../auth-layout'
import { UserAuthForm } from './components/user-auth-form'

export function SignIn() {
  const { t } = useTranslation()
  const { redirect } = useSearch({ from: '/(auth)/sign-in' })

  return (
    <AuthLayout>
      <div className='border-border bg-card w-full rounded-3xl border p-8 shadow-sm sm:p-10'>
        <div className='mb-8 space-y-2 text-center'>
          <h1 className='text-foreground text-3xl font-semibold tracking-tight'>
            智颖API监控
          </h1>
          <p className='text-muted-foreground text-sm'>{t('Sign in')}</p>
        </div>

        <UserAuthForm redirectTo={redirect} />
      </div>
    </AuthLayout>
  )
}
