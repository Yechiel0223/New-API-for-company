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
import { PieChart as PieChartIcon } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { IconBadge } from '@/components/ui/icon-badge'
import { useTheme } from '@/context/theme-provider'
import { MODEL_ANALYTICS_CHART_OPTIONS } from '@/features/dashboard/constants'
import {
  buildModelAnalyticsCharts,
  formatShanghaiRange,
} from '@/features/dashboard/lib/model-analytics'
import type {
  ModelAnalyticsChartTab,
  ModelAnalyticsData,
} from '@/features/dashboard/types'
import { VCHART_OPTION } from '@/lib/vchart'

interface ModelChartsProps {
  analytics: ModelAnalyticsData | undefined
  loading: boolean
  error: boolean
  onRetry: () => void
  defaultChartTab?: ModelAnalyticsChartTab
}

export function ModelCharts(props: ModelChartsProps) {
  const { t } = useTranslation()
  const { resolvedTheme } = useTheme()
  const [activeTab, setActiveTab] = useState<ModelAnalyticsChartTab>(
    props.defaultChartTab ?? 'trend'
  )
  const charts = useMemo(
    () =>
      buildModelAnalyticsCharts(props.analytics?.series ?? [], {
        granularity: props.analytics?.range.granularity ?? 'day',
        translate: t,
      }),
    [props.analytics, t]
  )
  const spec =
    activeTab === 'trend'
      ? charts.calls
      : activeTab === 'proportion'
        ? charts.proportion
        : charts.ranking
  const rangeText = props.analytics?.range
    ? formatShanghaiRange(props.analytics.range)
    : ''
  const tabDescription =
    activeTab === 'trend'
      ? t('Shows call changes across the selected range')
      : activeTab === 'proportion'
        ? t('Shows each model share by call count')
        : t('Ranks models by call count')

  return (
    <section className='overflow-hidden rounded-lg border'>
      <header className='bg-muted/20 flex flex-wrap items-center gap-3 border-b px-4 py-3'>
        <IconBadge tone='chart-4' size='sm'>
          <PieChartIcon />
        </IconBadge>
        <div className='min-w-0'>
          <div className='flex flex-wrap items-center gap-2'>
            <h3 className='text-sm font-semibold'>{t('Model Call Analytics')}</h3>
            <span className='text-muted-foreground text-xs'>
              {t('Total:')} {props.analytics?.summary.total_calls ?? 0}
            </span>
          </div>
          <p className='text-muted-foreground mt-1 truncate text-xs'>
            {rangeText ? `${tabDescription} · ${rangeText}` : tabDescription}
          </p>
        </div>
        <div className='ml-auto flex flex-wrap gap-1'>
          {MODEL_ANALYTICS_CHART_OPTIONS.map((option) => (
            <Button key={option.value} type='button' size='sm' variant={activeTab === option.value ? 'default' : 'ghost'} aria-pressed={activeTab === option.value} onClick={() => setActiveTab(option.value)}>
              {t(option.labelKey)}
            </Button>
          ))}
        </div>
      </header>
      <div className='h-[320px] px-3 py-4'>
        {props.loading ? (
          <div className='bg-muted/40 h-full animate-pulse rounded-md' />
        ) : props.error ? (
          <div className='flex h-full flex-col items-center justify-center gap-2 text-sm'><span>{t('Unable to load analytics')}</span><Button type='button' size='sm' onClick={props.onRetry}>{t('Retry')}</Button></div>
        ) : (props.analytics?.series.length ?? 0) === 0 ? (
          <div className='text-muted-foreground flex h-full items-center justify-center text-sm'>{t('No calls')}</div>
        ) : (
          <VChart spec={{ ...spec, theme: resolvedTheme === 'dark' ? 'dark' : 'light' }} option={VCHART_OPTION} />
        )}
      </div>
    </section>
  )
}
