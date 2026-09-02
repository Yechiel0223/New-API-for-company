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
import { RefreshCw } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  buildDefaultDashboardFilters,
  getSavedChartPreferences,
} from '@/features/dashboard/lib/filters'
import {
  createRangePreset,
  formatShanghaiRange,
  type ModelAnalyticsRangePreset,
} from '@/features/dashboard/lib/model-analytics'
import type { DashboardFilters } from '@/features/dashboard/types'
import { cn } from '@/lib/utils'

import { ModelsFilter } from './models-filter-dialog'

interface ModelAnalyticsToolbarProps {
  filters: DashboardFilters
  availableModels: string[]
  onFiltersChange: (filters: DashboardFilters) => void
  onRefresh: () => void
  refreshing: boolean
}

const PRESETS: Array<{
  value: ModelAnalyticsRangePreset
  label: string
}> = [
  { value: 'today', label: 'Today' },
  { value: '24h', label: '24 Hours' },
  { value: '7d', label: '7 Days' },
  { value: '30d', label: '30 Days' },
]

export function ModelAnalyticsToolbar(props: ModelAnalyticsToolbarProps) {
  const { t } = useTranslation()
  const [activePreset, setActivePreset] =
    useState<ModelAnalyticsRangePreset | null>('7d')
  const preferences = useMemo(() => getSavedChartPreferences(), [])
  const start = props.filters.start_timestamp
  const end = props.filters.end_timestamp
  const rangeLabel =
    start && end
      ? formatShanghaiRange({
          start: Math.floor(start.getTime() / 1000),
          end: Math.floor(end.getTime() / 1000),
        })
      : '--'

  const applyPreset = (preset: ModelAnalyticsRangePreset) => {
    const range = createRangePreset(preset)
    setActivePreset(preset)
    props.onFiltersChange({
      ...props.filters,
      start_timestamp: new Date(range.start * 1000),
      end_timestamp: new Date(range.end * 1000),
      time_granularity: range.granularity,
    })
  }

  return (
    <div className='rounded-lg border p-3 sm:p-4'>
      <div className='flex flex-wrap items-center gap-2'>
        <div
          className='bg-muted/60 flex flex-wrap gap-1 rounded-lg p-1'
          role='group'
          aria-label={t('Time range')}
        >
          {PRESETS.map((preset) => (
            <Button
              key={preset.value}
              type='button'
              size='sm'
              variant={activePreset === preset.value ? 'default' : 'ghost'}
              aria-pressed={activePreset === preset.value}
              onClick={() => applyPreset(preset.value)}
            >
              {t(preset.label)}
            </Button>
          ))}
        </div>
        <ModelsFilter
          preferences={preferences}
          currentFilters={props.filters}
          availableModels={props.availableModels}
          onFilterChange={(filters) => {
            setActivePreset(null)
            props.onFiltersChange(filters)
          }}
          onReset={() => {
            setActivePreset('7d')
            props.onFiltersChange(buildDefaultDashboardFilters(preferences))
          }}
          triggerLabelKey='Custom'
        />
        <Button
          type='button'
          size='sm'
          variant='outline'
          onClick={props.onRefresh}
          disabled={props.refreshing}
        >
          <RefreshCw
            className={cn('mr-2 size-4', props.refreshing && 'animate-spin')}
          />
          {t('Refresh')}
        </Button>
      </div>
      <div className='text-muted-foreground mt-2 text-xs' aria-live='polite'>
        {t('Current range:')} {rangeLabel} ·{' '}
        {t(
          props.filters.time_granularity === 'hour'
            ? 'Hour'
            : props.filters.time_granularity === 'week'
              ? 'Week'
              : 'Day'
        )}
      </div>
    </div>
  )
}
