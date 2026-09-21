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
import { zodResolver } from '@hookform/resolvers/zod'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import {
  Combobox,
  ComboboxInput,
  ComboboxContent,
  ComboboxList,
  ComboboxItem,
  ComboboxEmpty,
} from '@/components/ui/combobox'
import { Empty, EmptyHeader, EmptyTitle } from '@/components/ui/empty'
import {
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { useAuthStore } from '@/stores/auth-store'

import { getAnalyticsUsernames } from '../../api'
import { ModelAnalyticsSection } from '../models/model-analytics-section'

const userAnalyticsSchema = z.object({ username: z.string().trim().min(1) })

export function UserAnalyticsSection() {
  const user = useAuthStore((state) => state.auth.user)
  return (
    <UserAnalyticsForm
      key={user?.id}
      initialUsername={user?.username ?? ''}
      userId={user?.id}
    />
  )
}

function UserAnalyticsForm(props: {
  initialUsername: string
  userId?: number
}) {
  const { t } = useTranslation()
  const [username, setUsername] = useState(props.initialUsername)
  const [open, setOpen] = useState(false)
  const users = useQuery({
    queryKey: ['analytics-usernames', props.userId],
    queryFn: getAnalyticsUsernames,
    enabled: open && props.userId !== undefined,
    staleTime: 60_000,
  })
  const form = useForm<z.infer<typeof userAnalyticsSchema>>({
    resolver: zodResolver(userAnalyticsSchema),
    defaultValues: { username: props.initialUsername },
  })
  const invalid = Boolean(form.formState.errors.username)

  return (
    <div className='space-y-3 sm:space-y-4'>
      <form
        className='rounded-lg border p-3 sm:p-4'
        onSubmit={form.handleSubmit(
          (values) => {
            setUsername(values.username)
            setOpen(false)
          },
          () => setOpen(false)
        )}
      >
        <FieldGroup className='gap-3 sm:flex-row sm:items-start'>
          <Field className='min-w-0 flex-1' data-invalid={invalid}>
            <FieldLabel htmlFor='analytics-username'>
              {t('Username')}
            </FieldLabel>
            <Controller
              control={form.control}
              name='username'
              render={({ field }) => (
                <Combobox
                  items={users.data ?? []}
                  value={field.value || null}
                  modal={false}
                  inputValue={field.value}
                  onInputValueChange={field.onChange}
                  onValueChange={(value) => {
                    if (value !== null) field.onChange(value)
                  }}
                  open={open}
                  onOpenChange={setOpen}
                >
                  <ComboboxInput
                    id='analytics-username'
                    ref={field.ref}
                    onBlur={field.onBlur}
                    onFocus={() => {
                      if (!form.getFieldState('username').invalid) {
                        setOpen(true)
                      }
                    }}
                    placeholder={t('Enter username')}
                    aria-label={t('Username')}
                    aria-invalid={invalid}
                    aria-describedby={
                      invalid ? 'analytics-username-error' : undefined
                    }
                    showTrigger={false}
                    className='w-full'
                  />
                  <ComboboxContent>
                    {users.isFetching && (
                      <p className='p-3 text-sm' role='status'>
                        {t('Loading...')}
                      </p>
                    )}
                    {users.isError && (
                      <div className='p-3 text-sm' role='alert'>
                        <p>{t('Failed to load users')}</p>
                        <Button
                          type='button'
                          variant='outline'
                          size='sm'
                          className='mt-2'
                          onClick={() => void users.refetch()}
                        >
                          {t('Retry')}
                        </Button>
                      </div>
                    )}
                    {!users.isFetching && !users.isError && (
                      <ComboboxEmpty>{t('No results found')}</ComboboxEmpty>
                    )}
                    <ComboboxList>
                      {(name: string) => (
                        <ComboboxItem key={name} value={name}>
                          <span className='truncate'>{name}</span>
                        </ComboboxItem>
                      )}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              )}
            />
            {invalid && (
              <FieldError id='analytics-username-error'>
                {t('Enter username')}
              </FieldError>
            )}
          </Field>
          <Button type='submit' className='sm:mt-7'>
            {t('Search')}
          </Button>
        </FieldGroup>
      </form>
      {username ? (
        <>
          <p className='text-muted-foreground text-sm break-all' role='status'>
            {t('Viewing analytics for {{username}}', { username })}
          </p>
          <ModelAnalyticsSection key={username} username={username} />
        </>
      ) : (
        <Empty className='border'>
          <EmptyHeader>
            <EmptyTitle>{t('Enter a username to view analytics')}</EmptyTitle>
          </EmptyHeader>
        </Empty>
      )}
    </div>
  )
}
