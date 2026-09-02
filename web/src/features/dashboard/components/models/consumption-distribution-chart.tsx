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
import { VChart } from '@visactor/react-vchart'
import { AreaChart, BarChart3, WalletCards } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { IconBadge } from '@/components/ui/icon-badge'
import { useTheme } from '@/context/theme-provider'
import {
  buildModelAnalyticsCharts,
  formatShanghaiRange,
} from '@/features/dashboard/lib/model-analytics'
import type {
  ConsumptionDistributionChartType,
  ModelAnalyticsData,
} from '@/features/dashboard/types'
import { formatQuota } from '@/lib/format'
import { VCHART_OPTION } from '@/lib/vchart'

interface ConsumptionDistributionChartProps {
  analytics: ModelAnalyticsData | undefined
  loading: boolean
  error: boolean
  onRetry: () => void
  defaultChartType?: ConsumptionDistributionChartType
}

export function ConsumptionDistributionChart(
  props: ConsumptionDistributionChartProps
) {
  const { t } = useTranslation()
  const { resolvedTheme } = useTheme()
  const [chartType, setChartType] = useState<ConsumptionDistributionChartType>(
    props.defaultChartType ?? 'bar'
  )
  const charts = useMemo(
    () =>
      buildModelAnalyticsCharts(props.analytics?.series ?? [], {
        granularity: props.analytics?.range.granularity ?? 'day',
        translate: t,
      }),
    [props.analytics, t]
  )
  const baseSpec = charts.consumption
  const spec = {
    ...baseSpec,
    type: chartType,
    theme: resolvedTheme === 'dark' ? 'dark' : 'light',
  }
  const rangeText = props.analytics?.range
    ? formatShanghaiRange(props.analytics.range)
    : ''

  return (
    <section className='overflow-hidden rounded-lg border'>
      <header className='bg-muted/20 flex flex-wrap items-center gap-3 border-b px-4 py-3'>
        <IconBadge tone='success' size='sm'>
          <WalletCards />
        </IconBadge>
        <div className='min-w-0'>
          <div className='flex flex-wrap items-center gap-2'>
            <h3 className='text-sm font-semibold'>{t('Consumption trend')}</h3>
            <span className='text-muted-foreground text-xs'>
              {t('Total:')} {formatQuota(props.analytics?.summary.total_quota ?? 0)}
            </span>
          </div>
          <p className='text-muted-foreground mt-1 truncate text-xs'>
            {rangeText
              ? `${t('Selected range')}: ${rangeText}`
              : t('Grouped by model, shown in display currency')}
          </p>
        </div>
        <div className='ml-auto flex items-center gap-1'>
          <Button type='button' size='icon-sm' variant={chartType === 'bar' ? 'default' : 'ghost'} aria-label={t('Bar Chart')} onClick={() => setChartType('bar')}><BarChart3 /></Button>
          <Button type='button' size='icon-sm' variant={chartType === 'area' ? 'default' : 'ghost'} aria-label={t('Area Chart')} onClick={() => setChartType('area')}><AreaChart /></Button>
        </div>
      </header>
      <div className='h-[320px] px-3 py-4'>
        {props.loading ? (
          <div className='bg-muted/40 h-full animate-pulse rounded-md' />
        ) : props.error ? (
          <div className='flex h-full flex-col items-center justify-center gap-2 text-sm'>
            <span>{t('Unable to load analytics')}</span>
            <Button type='button' size='sm' onClick={props.onRetry}>{t('Retry')}</Button>
          </div>
        ) : (props.analytics?.series.length ?? 0) === 0 ? (
          <div className='text-muted-foreground flex h-full items-center justify-center text-sm'>{t('No calls')}</div>
        ) : (
          <VChart spec={spec} option={VCHART_OPTION} />
        )}
      </div>
    </section>
  )
}
