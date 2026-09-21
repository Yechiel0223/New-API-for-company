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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { getCurrencyDisplay, getCurrencyLabel } from '@/lib/currency'
import {
  formatQuota,
  parseQuotaFromDollars,
  quotaUnitsToDollars,
} from '@/lib/format'

import { adjustUserQuota } from '../api'

interface UserQuotaDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  userId: number
  currentQuota: number
  onSuccess: () => void
}

export function UserQuotaDialog(props: UserQuotaDialogProps) {
  return props.open ? <UserQuotaForm key={props.userId} {...props} /> : null
}

function UserQuotaForm(props: UserQuotaDialogProps) {
  const { t } = useTranslation()
  const [amount, setAmount] = useState(() =>
    String(quotaUnitsToDollars(props.currentQuota))
  )
  const [loading, setLoading] = useState(false)
  const { meta: currencyMeta } = getCurrencyDisplay()
  const currencyLabel = getCurrencyLabel()
  const tokensOnly = currencyMeta.kind === 'tokens'
  const amountValue = Number(amount)
  const quotaValue = parseQuotaFromDollars(amountValue)
  const isValid =
    amount.trim() !== '' &&
    Number.isFinite(amountValue) &&
    amountValue >= 0 &&
    Number.isSafeInteger(quotaValue) &&
    quotaValue >= 0

  const handleConfirm = async () => {
    if (loading || !isValid) return
    setLoading(true)
    try {
      const result = await adjustUserQuota({
        id: props.userId,
        action: 'add_quota',
        mode: 'override',
        value: quotaValue,
      })
      if (result.success) {
        toast.success(t('Quota adjusted successfully'))
        props.onOpenChange(false)
        props.onSuccess()
      } else {
        toast.error(result.message || t('Failed to adjust quota'))
      }
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('Failed to adjust quota'))
    } finally {
      setLoading(false)
    }
  }

  const placeholder = tokensOnly
    ? t('Enter quota in tokens')
    : t('Enter quota in {{currency}}', { currency: currencyLabel })

  return (
    <Dialog
      open={props.open}
      onOpenChange={(open) => {
        if (!loading) props.onOpenChange(open)
      }}
      title={t('Adjust Quota')}
      description={t(
        'Enter the new remaining quota to replace the current balance'
      )}
      contentHeight='auto'
      bodyClassName='space-y-4'
      footer={
        <>
          <Button
            variant='outline'
            disabled={loading}
            onClick={() => props.onOpenChange(false)}
          >
            {t('Cancel')}
          </Button>
          <Button onClick={handleConfirm} disabled={loading || !isValid}>
            {loading ? t('Processing...') : t('Confirm')}
          </Button>
        </>
      }
    >
      <p className='text-muted-foreground text-sm'>
        {t('Current quota')}: {formatQuota(props.currentQuota)}
        {isValid && ` → ${formatQuota(quotaValue)}`}
      </p>
      <div className='space-y-2'>
        <Label htmlFor='user-quota-amount'>
          {t('Remaining Quota ({{currency}})', { currency: currencyLabel })}
        </Label>
        <Input
          id='user-quota-amount'
          type='number'
          step={tokensOnly ? 1 : 0.000001}
          min={0}
          placeholder={placeholder}
          value={amount}
          disabled={loading}
          aria-invalid={!isValid}
          aria-describedby={!isValid ? 'user-quota-error' : undefined}
          onChange={(e) => setAmount(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault()
              void handleConfirm()
            }
          }}
        />
        {!isValid && (
          <p
            id='user-quota-error'
            role='alert'
            className='text-destructive text-sm'
          >
            {t('Enter a valid non-negative quota amount')}
          </p>
        )}
      </div>
    </Dialog>
  )
}
