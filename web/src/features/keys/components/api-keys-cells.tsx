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
import { Check, Copy, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { copyToClipboard } from '@/lib/copy-to-clipboard'

import type { ApiKey } from '../types'
import { useApiKeys } from './api-keys-provider'

export function ApiKeyCell(props: { apiKey: ApiKey }) {
  const { t } = useTranslation()
  const { resolveRealKey, loadingKeys, copiedKeyId, markKeyCopied } =
    useApiKeys()
  const isLoading = !!loadingKeys[props.apiKey.id]
  const isCopied = copiedKeyId === props.apiKey.id

  const handleCopy = async () => {
    const realKey = await resolveRealKey(props.apiKey.id)
    if (!realKey) return
    if (await copyToClipboard(realKey)) markKeyCopied(props.apiKey.id)
  }

  let copyIcon = <Copy className='size-3.5' />
  if (isLoading) copyIcon = <Loader2 className='size-3.5 animate-spin' />
  else if (isCopied) copyIcon = <Check className='size-3.5 text-green-600' />

  return (
    <div className='flex max-w-full min-w-0 items-center gap-1'>
      <span className='text-muted-foreground truncate font-mono text-xs'>
        sk-{props.apiKey.key}
      </span>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              variant='ghost'
              size='icon'
              className='size-7 shrink-0'
              aria-label={t('Copy API key')}
              onClick={handleCopy}
              disabled={isLoading}
            />
          }
        >
          {copyIcon}
        </TooltipTrigger>
        <TooltipContent>
          {isCopied ? t('Copied!') : t('Copy API key')}
        </TooltipContent>
      </Tooltip>
    </div>
  )
}
