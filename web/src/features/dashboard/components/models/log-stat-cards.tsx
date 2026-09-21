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
import { Coins, Hash, Layers3 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { IconBadge } from '@/components/ui/icon-badge'
import { Skeleton } from '@/components/ui/skeleton'
import type { ModelAnalyticsData } from '@/features/dashboard/types'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber, formatQuota } from '@/lib/format'

interface LogStatCardsProps {
  analytics: ModelAnalyticsData | undefined
  loading: boolean
  error: boolean
  onRetry: () => void
}

export function LogStatCards(props: LogStatCardsProps) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const summary = props.analytics?.summary
  const items = [
    {
      title: t('Total calls'),
      value: formatNumber(summary?.total_calls ?? 0, locale),
      description: summary
        ? `${t('Success')} ${summary.success_calls} · ${t('Failure')} ${summary.failure_calls} · ${t('Running')} ${summary.running_calls}`
        : t('Selected range'),
      icon: Hash,
      tone: 'info' as const,
    },
    {
      title: t('Total consumption'),
      value: formatQuota(summary?.total_quota ?? 0),
      description: t('Selected range'),
      icon: Coins,
      tone: 'success' as const,
    },
    {
      title: t('Total Token'),
      value: formatNumber(summary?.total_tokens ?? 0, locale),
      description: t('Selected range'),
      icon: Layers3,
      tone: 'chart-4' as const,
    },
  ]

  return (
    <div className='overflow-hidden rounded-lg border'>
      <div className='divide-border/60 grid grid-cols-1 divide-y sm:grid-cols-3 sm:divide-x sm:divide-y-0'>
        {items.map((item) => {
          const Icon = item.icon
          return (
            <div key={item.title} className='min-w-0 px-3 py-3 sm:px-5 sm:py-4'>
              <div className='flex items-center gap-2'>
                <IconBadge tone={item.tone} size='sm'>
                  <Icon />
                </IconBadge>
                <span className='text-muted-foreground truncate text-xs font-medium'>
                  {item.title}
                </span>
              </div>
              {props.loading ? (
                <Skeleton className='mt-2 h-7 w-24' />
              ) : (
                <div
                  className='mt-2 truncate font-mono text-xl font-bold tabular-nums'
                  title={props.error ? '--' : item.value}
                >
                  {props.error ? '--' : item.value}
                </div>
              )}
              <div className='text-muted-foreground mt-1 truncate text-xs'>
                {item.description}
              </div>
            </div>
          )
        })}
      </div>
      {props.error && (
        <div className='border-t p-2 text-center'>
          <Button
            type='button'
            size='sm'
            variant='ghost'
            onClick={props.onRetry}
          >
            {t('Retry')}
          </Button>
        </div>
      )}
    </div>
  )
}
