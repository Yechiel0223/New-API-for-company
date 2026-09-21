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
import { useId, useMemo, useRef, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'

import { createApiKey } from '../api'
import { ERROR_MESSAGES, SUCCESS_MESSAGES } from '../constants'
import {
  getApiKeyFormSchema,
  type ApiKeyFormValues,
  API_KEY_FORM_DEFAULT_VALUES,
  transformFormDataToPayload,
} from '../lib'
import { useApiKeys } from './api-keys-provider'

type ApiKeysCreateDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ApiKeysCreateDialog(props: ApiKeysCreateDialogProps) {
  const { t } = useTranslation()
  const { triggerRefresh } = useApiKeys()
  const nameId = useId()
  const pending = useRef(false)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const schema = useMemo(() => getApiKeyFormSchema(t), [t])
  const form = useForm<ApiKeyFormValues>({
    resolver: zodResolver(schema),
    defaultValues: API_KEY_FORM_DEFAULT_VALUES,
  })
  const name = form.watch('name')
  const error = form.formState.errors.name

  const handleOpenChange = (open: boolean) => {
    if (pending.current) return
    if (!open) form.reset()
    props.onOpenChange(open)
  }

  const onSubmit = async (data: ApiKeyFormValues) => {
    if (pending.current) return
    pending.current = true
    setIsSubmitting(true)
    try {
      const result = await createApiKey(transformFormDataToPayload(data))
      if (!result.success) {
        toast.error(result.message || t(ERROR_MESSAGES.CREATE_FAILED))
        return
      }
      toast.success(t(SUCCESS_MESSAGES.API_KEY_CREATED))
      form.reset()
      props.onOpenChange(false)
      triggerRefresh()
    } catch {
      toast.error(t(ERROR_MESSAGES.UNEXPECTED))
    } finally {
      pending.current = false
      setIsSubmitting(false)
    }
  }

  return (
    <Dialog open={props.open} onOpenChange={handleOpenChange}>
      <DialogContent
        className='rounded-2xl sm:max-w-sm'
        showCloseButton={!isSubmitting}
      >
        <DialogHeader>
          <DialogTitle>{t('Create API Key')}</DialogTitle>
          <DialogDescription className='sr-only'>
            {t('Enter a name')}
          </DialogDescription>
        </DialogHeader>
        <form
          onSubmit={(event) => {
            if (pending.current) {
              event.preventDefault()
              return
            }
            void form.handleSubmit(onSubmit)(event)
          }}
          className='grid gap-5'
          aria-busy={isSubmitting}
        >
          <FieldGroup>
            <Field data-invalid={!!error} data-disabled={isSubmitting}>
              <FieldLabel htmlFor={nameId} className='sr-only'>
                {t('Name')}
              </FieldLabel>
              <Input
                {...form.register('name')}
                id={nameId}
                placeholder={t('Enter a name')}
                className='rounded-full'
                disabled={isSubmitting}
                aria-invalid={!!error}
                aria-describedby={error ? `${nameId}-error` : undefined}
              />
              {error && <FieldError id={`${nameId}-error`} errors={[error]} />}
            </Field>
          </FieldGroup>
          <DialogFooter className='m-0 flex-row justify-end rounded-none border-0 bg-transparent p-0'>
            <Button
              type='button'
              variant='outline'
              className='rounded-full'
              disabled={isSubmitting}
              onClick={() => handleOpenChange(false)}
            >
              {t('Cancel')}
            </Button>
            <Button
              type='submit'
              className='rounded-full'
              disabled={isSubmitting || !name.trim()}
            >
              {isSubmitting ? t('Creating...') : t('Confirm')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
