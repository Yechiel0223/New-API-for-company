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
import { KeyRound, WalletCards } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { type SubmitErrorHandler, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  SideDrawerSection,
  SideDrawerSectionHeader,
  sideDrawerContentClassName,
  sideDrawerFooterClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
  sideDrawerSwitchItemClassName,
} from '@/components/drawer-layout'
import { Button } from '@/components/ui/button'
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
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Switch } from '@/components/ui/switch'
import { getCurrencyDisplay, getCurrencyLabel } from '@/lib/currency'

import { createApiKey, updateApiKey, getApiKey } from '../api'
import { ERROR_MESSAGES, SUCCESS_MESSAGES } from '../constants'
import {
  getApiKeyFormSchema,
  type ApiKeyFormValues,
  getApiKeyFormDefaultValues,
  transformFormDataToPayload,
  transformApiKeyToFormDefaults,
} from '../lib'
import type { ApiKey } from '../types'
import { useApiKeys } from './api-keys-provider'

type ApiKeyMutateDrawerProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  currentRow?: ApiKey
}

export function ApiKeysMutateDrawer({
  open,
  onOpenChange,
  currentRow,
}: ApiKeyMutateDrawerProps) {
  const { t } = useTranslation()
  const isUpdate = !!currentRow
  const currentRowId = currentRow?.id
  const { triggerRefresh } = useApiKeys()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [initializedTarget, setInitializedTarget] = useState<string | null>(
    null
  )
  const {
    data: apiKeyData,
    isFetched: apiKeyFetched,
    isFetching: apiKeyFetching,
  } = useQuery({
    queryKey: ['api-key', currentRowId],
    queryFn: () => getApiKey(currentRowId ?? 0),
    enabled: open && isUpdate && currentRowId !== undefined,
    staleTime: 0,
  })

  const schema = useMemo(() => getApiKeyFormSchema(t), [t])

  const form = useForm<ApiKeyFormValues>({
    resolver: zodResolver(schema),
    defaultValues: getApiKeyFormDefaultValues(false),
  })

  // Load existing data when updating
  useEffect(() => {
    if (!open) {
      setInitializedTarget(null)
      return
    }
    if (isUpdate && (!apiKeyFetched || apiKeyFetching)) return

    const target = isUpdate && currentRow ? `update:${currentRow.id}` : 'create'
    if (initializedTarget === target) return
    if (isUpdate && currentRow) {
      if (apiKeyData?.success && apiKeyData.data) {
        form.reset(transformApiKeyToFormDefaults(apiKeyData.data))
        setInitializedTarget(target)
      }
    } else {
      form.reset(getApiKeyFormDefaultValues(false))
      setInitializedTarget(target)
    }
  }, [
    open,
    isUpdate,
    currentRow,
    form,
    apiKeyData,
    apiKeyFetched,
    apiKeyFetching,
    initializedTarget,
  ])

  const formTarget =
    isUpdate && currentRow ? `update:${currentRow.id}` : 'create'
  const isFormInitialized = initializedTarget === formTarget
  const onSubmit = async (data: ApiKeyFormValues) => {
    if (!isFormInitialized || isSubmitting) return
    setIsSubmitting(true)
    try {
      const basePayload = transformFormDataToPayload(data)

      if (isUpdate && currentRow) {
        const result = await updateApiKey({
          ...basePayload,
          id: currentRow.id,
        })
        if (result.success) {
          toast.success(t(SUCCESS_MESSAGES.API_KEY_UPDATED))
          onOpenChange(false)
          triggerRefresh()
        } else {
          toast.error(result.message || t(ERROR_MESSAGES.UPDATE_FAILED))
        }
      } else {
        // Create mode - handle batch creation
        const count = data.tokenCount || 1
        let successCount = 0

        for (let i = 0; i < count; i++) {
          const result = await createApiKey({
            ...basePayload,
            name:
              i === 0 && data.name
                ? data.name
                : `${data.name || 'default'}-${Math.random().toString(36).slice(2, 8)}`,
          })
          if (result.success) {
            successCount++
          } else {
            toast.error(result.message || t(ERROR_MESSAGES.CREATE_FAILED))
            break
          }
        }

        if (successCount > 0) {
          toast.success(
            t('Successfully created {{count}} API Key(s)', {
              count: successCount,
            })
          )
          onOpenChange(false)
          triggerRefresh()
        }
      }
    } catch {
      toast.error(t(ERROR_MESSAGES.UNEXPECTED))
    } finally {
      setIsSubmitting(false)
    }
  }

  const onInvalid: SubmitErrorHandler<ApiKeyFormValues> = () => {
    toast.error(t('Please fix the highlighted fields before saving'))
  }

  const { meta: currencyMeta } = getCurrencyDisplay()
  const currencyLabel = getCurrencyLabel()
  const tokensOnly = currencyMeta.kind === 'tokens'
  const quotaLabel = t('Quota ({{currency}})', { currency: currencyLabel })
  const quotaPlaceholder = tokensOnly
    ? t('Enter quota in tokens')
    : t('Enter quota in {{currency}}', { currency: currencyLabel })
  const unlimitedQuota = form.watch('unlimited_quota')

  return (
    <Sheet
      open={open}
      onOpenChange={(v) => {
        onOpenChange(v)
        if (!v) {
          form.reset()
        }
      }}
    >
      <SheetContent
        className={sideDrawerContentClassName('max-w-none sm:!max-w-[620px]')}
      >
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>
            {isUpdate ? t('Update API Key') : t('Create API Key')}
          </SheetTitle>
          <SheetDescription>
            {isUpdate
              ? t('Update the API key by providing necessary info.')
              : t('Add a new API key by providing necessary info.')}
          </SheetDescription>
        </SheetHeader>
        <Form {...form}>
          <form
            id='api-key-form'
            onSubmit={form.handleSubmit(onSubmit, onInvalid)}
            aria-busy={!isFormInitialized}
            inert={!isFormInitialized || isSubmitting ? true : undefined}
            className={sideDrawerFormClassName('gap-5')}
          >
            <SideDrawerSection>
              <SideDrawerSectionHeader
                title={t('Basic Information')}
                description={t('Set API key basic information')}
                icon={<KeyRound className='size-4' />}
                iconTone='info'
              />
              <FormField
                control={form.control}
                name='name'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Name')}</FormLabel>
                    <FormControl>
                      <Input {...field} placeholder={t('Enter a name')} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {!isUpdate && (
                <FormField
                  control={form.control}
                  name='tokenCount'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Quantity')}</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          type='number'
                          min='1'
                          placeholder={t('Number of keys to create')}
                          onChange={(e) =>
                            field.onChange(
                              Number.parseInt(e.target.value, 10) || 1
                            )
                          }
                        />
                      </FormControl>
                      <FormDescription>
                        {t(
                          'Create multiple API keys at once (random suffix will be added to names)'
                        )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}
            </SideDrawerSection>

            <SideDrawerSection>
              <SideDrawerSectionHeader
                title={t('Quota Settings')}
                description={t('Set quota amount and limits')}
                icon={<WalletCards className='size-4' />}
                iconTone='success'
              />
              {!unlimitedQuota && (
                <FormField
                  control={form.control}
                  name='remain_quota_dollars'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{quotaLabel}</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          type='number'
                          step={tokensOnly ? 1 : 0.01}
                          placeholder={quotaPlaceholder}
                          onChange={(e) =>
                            field.onChange(
                              Number.parseFloat(e.target.value) || 0
                            )
                          }
                        />
                      </FormControl>
                      <FormDescription>
                        {tokensOnly
                          ? t('Enter the quota amount in tokens')
                          : t('Enter the quota amount in {{currency}}', {
                              currency: currencyLabel,
                            })}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}

              <FormField
                control={form.control}
                name='unlimited_quota'
                render={({ field }) => (
                  <FormItem className={sideDrawerSwitchItemClassName()}>
                    <div className='flex flex-col gap-0.5'>
                      <FormLabel className='text-sm'>
                        {t('Unlimited Quota')}
                      </FormLabel>
                      <FormDescription className='text-xs'>
                        {t('Enable unlimited quota for this API key')}
                      </FormDescription>
                    </div>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                  </FormItem>
                )}
              />
            </SideDrawerSection>
          </form>
        </Form>
        <SheetFooter className={sideDrawerFooterClassName()}>
          <SheetClose
            render={<Button variant='outline' className='w-full sm:w-auto' />}
          >
            {t('Close')}
          </SheetClose>
          <Button
            type='button'
            onClick={form.handleSubmit(onSubmit, onInvalid)}
            disabled={!isFormInitialized || isSubmitting}
            className='w-full sm:w-auto'
          >
            {isSubmitting ? t('Saving...') : t('Save changes')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
