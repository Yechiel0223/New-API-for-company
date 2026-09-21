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
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  SideDrawerSection,
  sideDrawerContentClassName,
  sideDrawerFooterClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import {
  ADMIN_PERMISSION_ACTIONS,
  ADMIN_PERMISSION_RESOURCES,
  EMPTY_PERMISSION_CATALOG,
  hasPermission,
  normalizeAdminPermissions,
} from '@/lib/admin-permissions'
import { getCurrencyDisplay, getCurrencyLabel } from '@/lib/currency'
import { parseQuotaFromDollars } from '@/lib/format'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import {
  createUser,
  updateUser,
  getUser,
  getPermissionCatalog,
  adjustUserQuota,
} from '../api'
import { ERROR_MESSAGES, SUCCESS_MESSAGES } from '../constants'
import {
  userFormSchema,
  type UserFormValues,
  USER_FORM_DEFAULT_VALUES,
  transformFormDataToPayload,
  transformUserToFormDefaults,
} from '../lib'
import type { User } from '../types'
import { useUsers } from './users-provider'

type UsersMutateDrawerProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  currentRow?: User
}

export function UsersMutateDrawer({
  open,
  onOpenChange,
  currentRow,
}: UsersMutateDrawerProps) {
  const { t } = useTranslation()
  const isUpdate = !!currentRow
  const { triggerRefresh } = useUsers()
  const currentUser = useAuthStore((s) => s.auth.user)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [loadedUser, setLoadedUser] = useState<User | null>(null)

  // Permission catalog is owned by the backend; fetched once and reused.
  const { data: permissionCatalog = EMPTY_PERMISSION_CATALOG } = useQuery({
    queryKey: ['admin-permission-catalog'],
    queryFn: getPermissionCatalog,
    staleTime: 5 * 60 * 1000,
  })

  const form = useForm<UserFormValues>({
    resolver: zodResolver(userFormSchema),
    defaultValues: USER_FORM_DEFAULT_VALUES,
  })

  // Load existing data when updating
  useEffect(() => {
    let active = true
    setLoadedUser(null)
    if (open && isUpdate && currentRow) {
      // For update, fetch fresh data
      getUser(currentRow.id)
        .then((result) => {
          if (active && result.success && result.data) {
            setLoadedUser(result.data)
            form.reset(transformUserToFormDefaults(result.data))
          }
        })
        .catch(() => {
          if (active) toast.error(t('Failed to load users'))
        })
    } else if (open && !isUpdate) {
      // For create, reset to defaults
      form.reset(USER_FORM_DEFAULT_VALUES)
    }
    return () => {
      active = false
    }
  }, [open, isUpdate, currentRow, form, t])

  const { meta: currencyMeta } = getCurrencyDisplay()
  const currencyLabel = getCurrencyLabel()
  const tokensOnly = currencyMeta.kind === 'tokens'

  const userDataReady = !isUpdate || loadedUser?.id === currentRow?.id
  const dirtyFields = form.formState.dirtyFields
  const selectedRole = form.watch('role')
  const canEditAdminPermissions = currentUser?.role === ROLE.SUPER_ADMIN
  const targetIsAdmin = (selectedRole ?? currentRow?.role ?? 0) >= ROLE.ADMIN

  const onSubmit = async (data: UserFormValues) => {
    if (!userDataReady || isSubmitting) return
    if (isUpdate && data.quota_dollars == null) {
      form.setError('quota_dollars', {
        type: 'required',
        message: t('Enter a valid non-negative quota amount'),
      })
      return
    }
    if (!isUpdate) {
      const passwordLength = data.password?.length || 0
      if (passwordLength < 8 || passwordLength > 20) {
        form.setError('password', {
          type: 'manual',
          message: t('Password must be between 8 and 20 characters'),
        })
        return
      }
    }

    const quotaChanged = isUpdate && Boolean(dirtyFields.quota_dollars)
    const detailsChanged = Boolean(
      dirtyFields.password ||
      dirtyFields.admin_permissions ||
      (isUpdate && loadedUser?.group !== 'default')
    )
    let savingQuota = false
    let detailsSaved = false
    setIsSubmitting(true)
    try {
      if (!isUpdate || detailsChanged) {
        const payload = transformFormDataToPayload(
          data,
          currentRow?.id,
          permissionCatalog
        )
        const result = isUpdate
          ? await updateUser(payload as typeof payload & { id: number })
          : await createUser(payload)
        if (!result.success) {
          toast.error(
            result.message ||
              t(
                isUpdate
                  ? ERROR_MESSAGES.UPDATE_FAILED
                  : ERROR_MESSAGES.CREATE_FAILED
              )
          )
          return
        }
        if (isUpdate) {
          detailsSaved = true
          setLoadedUser((user) => (user ? { ...user, group: 'default' } : user))
          form.resetField('password', { defaultValue: '' })
          form.resetField('group', { defaultValue: data.group })
          form.resetField('admin_permissions', {
            defaultValue: data.admin_permissions,
          })
          triggerRefresh()
        }
      }
      if (currentRow && quotaChanged) {
        savingQuota = true
        const result = await adjustUserQuota({
          id: currentRow.id,
          action: 'add_quota',
          mode: 'override',
          value: parseQuotaFromDollars(data.quota_dollars ?? 0),
        })
        if (!result.success) {
          throw new Error(result.message || t('Failed to adjust quota'))
        }
      }
      toast.success(
        t(
          isUpdate
            ? SUCCESS_MESSAGES.USER_UPDATED
            : SUCCESS_MESSAGES.USER_CREATED
        )
      )
      onOpenChange(false)
      triggerRefresh()
    } catch {
      if (savingQuota) {
        form.setError('quota_dollars', {
          type: 'server',
          message: detailsSaved
            ? t(
                'User details saved, but quota update failed. Save again to retry the quota update.'
              )
            : t('Failed to adjust quota'),
        })
      } else {
        toast.error(t(ERROR_MESSAGES.UNEXPECTED))
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Sheet
      open={open}
      onOpenChange={(v) => {
        if (isSubmitting) return
        onOpenChange(v)
        if (!v) {
          form.reset()
        }
      }}
    >
      <SheetContent className={sideDrawerContentClassName('sm:max-w-[600px]')}>
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>
            {isUpdate ? t('Update') : t('Create')} {t('User')}
          </SheetTitle>
          <SheetDescription>
            {isUpdate
              ? t('Update the user by providing necessary info.')
              : t('Add a new user by providing necessary info.')}
          </SheetDescription>
        </SheetHeader>
        <Form {...form}>
          <form
            id='user-form'
            noValidate
            onSubmit={form.handleSubmit(onSubmit)}
            className={sideDrawerFormClassName()}
          >
            {/* Basic Information */}
            <SideDrawerSection>
              <h3 className='text-sm font-medium'>{t('Basic Information')}</h3>

              <FormField
                control={form.control}
                name='username'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Username')}</FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        placeholder={t('Enter username')}
                        disabled={isUpdate}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {!isUpdate && (
                <FormField
                  control={form.control}
                  name='role'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Role')}</FormLabel>
                      <Select
                        items={[
                          { value: '1', label: t('Common User') },
                          { value: '10', label: t('Admin') },
                        ]}
                        onValueChange={(value) =>
                          value !== null &&
                          field.onChange(Number.parseInt(value))
                        }
                        value={String(field.value)}
                      >
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue placeholder={t('Select a role')} />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent alignItemWithTrigger={false}>
                          <SelectGroup>
                            <SelectItem value='1'>
                              {t('Common User')}
                            </SelectItem>
                            <SelectItem value='10'>{t('Admin')}</SelectItem>
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                      <FormDescription>
                        {t("Set the user's role (cannot be Root)")}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}

              <FormField
                control={form.control}
                name='password'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Password')}</FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        type='password'
                        placeholder={
                          isUpdate
                            ? t('Leave empty to keep unchanged')
                            : t('Enter password (8-20 characters)')
                        }
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </SideDrawerSection>

            {/* Group & Quota Settings (Update only) */}
            {isUpdate && (
              <SideDrawerSection>
                <h3 className='text-sm font-medium'>{t('Quota Settings')}</h3>

                <FormField
                  control={form.control}
                  name='quota_dollars'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('Remaining Quota ({{currency}})', {
                          currency: currencyLabel,
                        })}
                      </FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          type='number'
                          step={tokensOnly ? 1 : 0.000001}
                          min={0}
                          value={field.value ?? ''}
                          onChange={(event) =>
                            field.onChange(
                              event.target.value === ''
                                ? null
                                : event.target.valueAsNumber
                            )
                          }
                          disabled={!userDataReady || isSubmitting}
                        />
                      </FormControl>
                      <FormDescription>
                        {t(
                          'Enter the new remaining quota to replace the current balance'
                        )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </SideDrawerSection>
            )}

            {canEditAdminPermissions &&
              targetIsAdmin &&
              permissionCatalog.resources.length > 0 && (
                <SideDrawerSection>
                  <h3 className='text-sm font-medium'>
                    {t('Admin Permissions')}
                  </h3>
                  <p className='text-muted-foreground text-xs'>
                    {t(
                      'Default administrator permissions can be overridden for this user.'
                    )}
                  </p>
                  <FormField
                    control={form.control}
                    name='admin_permissions'
                    render={({ field }) => {
                      const selected = normalizeAdminPermissions(
                        field.value,
                        permissionCatalog
                      )
                      return (
                        <FormItem>
                          <div className='space-y-3'>
                            {permissionCatalog.resources.map((resource) => (
                              <div
                                key={resource.resource}
                                className='space-y-2 rounded-md border p-3'
                              >
                                <div className='text-sm font-medium'>
                                  {t(resource.label_key)}
                                </div>
                                <div className='space-y-2'>
                                  {resource.actions.map((option) => (
                                    <label
                                      key={option.action}
                                      className='flex items-start gap-3'
                                    >
                                      <Checkbox
                                        checked={
                                          selected[resource.resource]?.[
                                            option.action
                                          ] === true
                                        }
                                        onCheckedChange={(checked) => {
                                          field.onChange({
                                            ...selected,
                                            [resource.resource]: {
                                              ...selected[resource.resource],
                                              [option.action]: checked === true,
                                            },
                                          })
                                        }}
                                      />
                                      <span className='flex flex-col gap-1'>
                                        <span className='text-sm font-medium'>
                                          {t(option.label_key)}
                                        </span>
                                        <span className='text-muted-foreground text-xs'>
                                          {t(option.description_key)}
                                        </span>
                                      </span>
                                    </label>
                                  ))}
                                </div>
                              </div>
                            ))}
                          </div>
                          <FormMessage />
                        </FormItem>
                      )
                    }}
                  />
                  {currentUser && (
                    <p className='text-muted-foreground text-xs'>
                      {hasPermission(
                        currentUser,
                        ADMIN_PERMISSION_RESOURCES.CHANNEL,
                        ADMIN_PERMISSION_ACTIONS.SENSITIVE_WRITE
                      )
                        ? t('Your account can edit sensitive channel settings.')
                        : t(
                            'Your account cannot edit sensitive channel settings.'
                          )}
                    </p>
                  )}
                </SideDrawerSection>
              )}
          </form>
        </Form>
        <SheetFooter className={sideDrawerFooterClassName()}>
          <SheetClose
            render={<Button variant='outline' disabled={isSubmitting} />}
          >
            {t('Close')}
          </SheetClose>
          <Button
            form='user-form'
            type='submit'
            disabled={isSubmitting || !userDataReady}
          >
            {isSubmitting ? t('Saving...') : t('Save changes')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
