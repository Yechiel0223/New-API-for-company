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
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import {
  formatShanghaiBucket,
  shortModelName,
} from '@/features/dashboard/lib/model-analytics'
import type {
  ModelHealthData,
  ModelHealthRow,
} from '@/features/dashboard/types'
import { cn } from '@/lib/utils'

interface PerformanceOverviewProps {
  data: ModelHealthData | undefined
  loading: boolean
  error: boolean
  onRetry: () => void
  onModelClick: (model: ModelHealthRow) => void
}

function formatDuration(value: number | null): string {
  if (value == null) return 'Sample shortage'
  const seconds = Math.round(value / 1000)
  if (seconds < 60) return `${seconds}s`
  return `${Math.floor(seconds / 60)}m${seconds % 60}s`
}

export function PerformanceOverview(props: PerformanceOverviewProps) {
  const { t } = useTranslation()
  const overall = props.data?.overall
  const statusLabel =
    overall?.status === 'healthy'
      ? t('Healthy')
      : overall?.status === 'warning'
        ? t('Warning')
        : overall?.status === 'fault'
          ? t('Fault')
          : t('No calls')
  const completed =
    (overall?.success_calls ?? 0) + (overall?.failure_calls ?? 0)

  return (
    <section
      className='overflow-hidden rounded-lg border'
      aria-label={t('Performance health')}
    >
      <div className='bg-muted/20 flex flex-wrap items-center gap-2 border-b px-4 py-3'>
        <div className='mr-auto'>
          <h3 className='text-sm font-semibold'>
            {t('Performance health')}
          </h3>
          <p className='text-muted-foreground mt-1 text-xs'>
            {t('Follows the selected dashboard range')}
            {props.data?.window_hours ? ` · ${props.data.window_hours}h` : ''}
          </p>
        </div>
      </div>
      {props.loading ? (
        <div className='space-y-2 p-4'>
          <Skeleton className='h-6 w-full' />
          <Skeleton className='h-12 w-full' />
        </div>
      ) : props.error ? (
        <div className='p-4 text-center text-sm'>
          <div>{t('Unable to load analytics')}</div>
          <Button type='button' size='sm' variant='ghost' onClick={props.onRetry}>
            {t('Retry')}
          </Button>
        </div>
      ) : (
        <>
          <div className='text-muted-foreground grid gap-x-4 gap-y-2 px-4 py-3 text-xs sm:grid-cols-[minmax(11rem,1fr)_9.5rem_5rem_5rem_6rem_minmax(9rem,auto)] sm:items-center'>
            <span
              className={cn(
                'font-medium tabular-nums',
                overall?.status === 'healthy' && 'text-success',
                overall?.status === 'warning' && 'text-warning',
                overall?.status === 'fault' && 'text-destructive'
              )}
            >
              {statusLabel}
            </span>
            <span className='tabular-nums'>
              {t('Success')} {overall?.success_calls ?? 0}/{completed} ·{' '}
              {overall?.success_rate ?? 0}%
            </span>
            <span className='tabular-nums'>
              {t('Failure')} {overall?.failure_calls ?? 0}
            </span>
            <span className='tabular-nums'>
              {t('Running')} {overall?.running_calls ?? 0}
            </span>
            <span className='tabular-nums'>
              {t('Stuck')} {overall?.stuck_calls ?? 0}
            </span>
            <span className='tabular-nums'>
              {t('Last updated:')}{' '}
              {props.data
                ? formatShanghaiBucket(props.data.updated_at, 'hour')
                : '--'}
            </span>
          </div>
          <div className='divide-y'>
            {props.data?.models.map((model) => (
              <button
                key={model.model_name}
                type='button'
                className='hover:bg-muted/40 grid w-full gap-2 px-4 py-3 text-left sm:grid-cols-[minmax(11rem,1fr)_9.5rem_7rem_9rem] sm:items-center sm:gap-4'
                onClick={() => props.onModelClick(model)}
                title={model.model_name}
              >
                <span className='truncate font-medium'>
                  {shortModelName(model.model_name)}
                </span>
                <span className='text-muted-foreground text-sm tabular-nums'>
                  {t('Success')} {model.success_calls}/
                  {model.success_calls + model.failure_calls} ·{' '}
                  {model.success_rate}%
                </span>
                <span
                  className='text-muted-foreground text-sm tabular-nums'
                  title={
                    model.successful_duration_samples < 20
                      ? t('Small sample, for reference only')
                      : undefined
                  }
                >
                  P50 {formatDuration(model.p50_ms)}
                </span>
                <span
                  className='text-muted-foreground text-sm tabular-nums'
                  title={
                    model.successful_duration_samples < 20
                      ? t('Small sample, for reference only')
                      : undefined
                  }
                >
                  P95 {formatDuration(model.p95_ms)}
                </span>
              </button>
            ))}
            {props.data?.models.length === 0 && (
              <div className='text-muted-foreground p-6 text-center text-sm'>
                {t('No calls')}
              </div>
            )}
          </div>
        </>
      )}
    </section>
  )
}
