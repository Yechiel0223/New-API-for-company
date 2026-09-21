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
import type { Row } from '@tanstack/react-table'
import { Trash2, PowerOff, Loader2 } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import { updateApiKeyStatus } from '../api'
import { API_KEY_STATUS, ERROR_MESSAGES, SUCCESS_MESSAGES } from '../constants'
import { apiKeySchema } from '../types'
import { useApiKeys } from './api-keys-provider'

type DataTableRowActionsProps<TData> = { row: Row<TData> }

export function DataTableRowActions<TData>(
  props: DataTableRowActionsProps<TData>
) {
  const { t } = useTranslation()
  const apiKey = apiKeySchema.parse(props.row.original)
  const { setOpen, setCurrentRow, triggerRefresh } = useApiKeys()
  const [isDisabling, setIsDisabling] = useState(false)

  const handleDisable = async () => {
    if (isDisabling || apiKey.status !== API_KEY_STATUS.ENABLED) return
    setIsDisabling(true)
    try {
      const result = await updateApiKeyStatus(
        apiKey.id,
        API_KEY_STATUS.DISABLED
      )
      if (result.success) {
        toast.success(t(SUCCESS_MESSAGES.API_KEY_DISABLED))
        triggerRefresh()
      } else {
        toast.error(result.message || t(ERROR_MESSAGES.STATUS_UPDATE_FAILED))
      }
    } catch {
      toast.error(t(ERROR_MESSAGES.UNEXPECTED))
    } finally {
      setIsDisabling(false)
    }
  }

  return (
    <div className='flex items-center gap-1'>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              variant='ghost'
              size='icon-sm'
              aria-label={t('Disable')}
              onClick={handleDisable}
              disabled={isDisabling || apiKey.status !== API_KEY_STATUS.ENABLED}
            />
          }
        >
          {isDisabling ? (
            <Loader2 className='size-4 animate-spin' />
          ) : (
            <PowerOff className='size-4' />
          )}
        </TooltipTrigger>
        <TooltipContent>{t('Disable')}</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              variant='ghost'
              size='icon-sm'
              aria-label={t('Delete')}
              className='text-destructive hover:text-destructive'
              onClick={() => {
                setCurrentRow(apiKey)
                setOpen('delete')
              }}
            />
          }
        >
          <Trash2 className='size-4' />
        </TooltipTrigger>
        <TooltipContent>{t('Delete')}</TooltipContent>
      </Tooltip>
    </div>
  )
}
