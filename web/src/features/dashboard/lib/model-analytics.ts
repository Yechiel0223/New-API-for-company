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
import type { IChartSpec } from '@visactor/vchart'

import type {
  AnalyticsBucket,
  AnalyticsGranularity,
} from '@/features/dashboard/types'
import dayjs from '@/lib/dayjs'
import {
  formatQuota,
  parseQuotaFromDollars,
  quotaUnitsToDollars,
} from '@/lib/format'

const SHANGHAI_TIMEZONE = 'Asia/Shanghai'

export type ModelAnalyticsRangePreset = 'today' | '24h' | '7d' | '30d'

export interface UnixAnalyticsRange {
  start: number
  end: number
  granularity: AnalyticsGranularity
}

const MODEL_SHORT_NAMES: Record<string, string> = {
  'doubao-seedance-2-5-260628': 'Seedance 2.5',
  'doubao-seedance-2-0-260128': 'Seedance 2.0',
  'doubao-seedream-5-0-pro-260628': 'Seedream 5.0 Pro',
}

export function shortModelName(modelId: string): string {
  return MODEL_SHORT_NAMES[modelId] ?? modelId
}

export function createRangePreset(
  preset: ModelAnalyticsRangePreset,
  now: Date = new Date()
): UnixAnalyticsRange {
  const end = Math.floor(now.getTime() / 1000)
  if (preset === 'today') {
    return {
      start: dayjs(now).tz(SHANGHAI_TIMEZONE).startOf('day').unix(),
      end,
      granularity: 'hour',
    }
  }

  const days = preset === '24h' ? 1 : preset === '7d' ? 7 : 30
  return {
    start: end - days * 86_400,
    end,
    granularity: days === 1 ? 'hour' : 'day',
  }
}

export function formatShanghaiRange(range: {
  start: number
  end: number
}): string {
  const format = (timestamp: number) =>
    dayjs.unix(timestamp).tz(SHANGHAI_TIMEZONE).format('YYYY-MM-DD HH:mm')
  return `${format(range.start)} – ${format(range.end)}`
}

export function formatShanghaiBucket(
  timestamp: number,
  granularity: AnalyticsGranularity
): string {
  const value = dayjs.unix(timestamp).tz(SHANGHAI_TIMEZONE)
  if (granularity === 'hour') return value.format('MM-DD HH:mm')
  if (granularity === 'week') return value.format('YYYY-MM-DD')
  return value.format('MM-DD')
}

type ChartDatum = {
  bucketStart: number
  bucketEnd: number
  timeLabel: string
  model: string
  modelId: string
  calls: number
  success: number
  failure: number
  quota: number
  quotaAmount: number
  tokens: number
  imageCount: number
  videoCount: number
}

export interface ModelAnalyticsChartOptions {
  granularity: AnalyticsGranularity
  translate?: (key: string) => string
}

export interface ModelAnalyticsChartSpecs {
  consumption: IChartSpec & {
    data: Array<{ id: string; values: ChartDatum[] }>
  }
  calls: IChartSpec
  proportion: IChartSpec
  ranking: IChartSpec
  summary: { totalQuota: number; totalCalls: number }
}

export function buildModelAnalyticsCharts(
  buckets: AnalyticsBucket[],
  options: ModelAnalyticsChartOptions
): ModelAnalyticsChartSpecs {
  const t = options.translate ?? ((key: string) => key)
  const values: ChartDatum[] = buckets.map((bucket) => ({
    bucketStart: bucket.bucket_start,
    bucketEnd: bucket.bucket_end,
    timeLabel: formatShanghaiBucket(bucket.bucket_start, options.granularity),
    model: shortModelName(bucket.model_name),
    modelId: bucket.model_name,
    calls: bucket.total_calls,
    success: bucket.success_calls,
    failure: bucket.failure_calls,
    quota: bucket.quota,
    quotaAmount: quotaUnitsToDollars(bucket.quota),
    tokens: bucket.tokens,
    imageCount: bucket.output_counts.image ?? 0,
    videoCount: bucket.output_counts.video ?? 0,
  }))
  const timeAxis = {
    orient: 'bottom' as const,
    type: 'band' as const,
    label: {
      autoHide: true,
      autoRotate: true,
      autoLimit: true,
      style: { fontSize: 11 },
    },
  }
  const leftAxis = {
    orient: 'left' as const,
    type: 'linear' as const,
    label: {
      autoLimit: true,
      style: { fontSize: 11 },
    },
  }
  const amountAxis = {
    ...leftAxis,
    label: {
      ...leftAxis.label,
      formatMethod: (value: number) => formatQuota(parseQuotaFromDollars(value)),
    },
  }
  const axes = [
    timeAxis,
    leftAxis,
  ]
  const amountAxes = [
    timeAxis,
    {
      ...amountAxis,
    },
  ]
  const legends = { visible: true, selectMode: 'multiple' as const }
  const totals = new Map<string, { calls: number; quota: number }>()
  for (const row of values) {
    const current = totals.get(row.modelId) ?? { calls: 0, quota: 0 }
    current.calls += row.calls
    current.quota += row.quota
    totals.set(row.modelId, current)
  }
  const modelTotals = Array.from(totals.entries()).map(([modelId, total]) => ({
    model: shortModelName(modelId),
    modelId,
    calls: total.calls,
    quota: total.quota,
  }))
  const totalQuota = values.reduce((sum, row) => sum + row.quota, 0)
  const totalCalls = values.reduce((sum, row) => sum + row.calls, 0)

  return {
    consumption: {
      type: 'bar',
      data: [{ id: 'model-consumption', values }],
      xField: 'timeLabel',
      yField: 'quotaAmount',
      seriesField: 'model',
      stack: true,
      axes: amountAxes,
      legends,
      tooltip: {
        mark: {
          content: [
            { key: t('Time'), value: (datum?: ChartDatum) => datum?.timeLabel ?? '' },
            { key: t('Model'), value: (datum?: ChartDatum) => datum?.model ?? '' },
            { key: t('Full model ID'), value: (datum?: ChartDatum) => datum?.modelId ?? '' },
            { key: t('Calls'), value: (datum?: ChartDatum) => datum?.calls ?? 0 },
            {
              key: t('Consumption'),
              value: (datum?: ChartDatum) => formatQuota(datum?.quota ?? 0),
            },
            {
              key: t('Tokens'),
              value: (datum?: ChartDatum) =>
                (datum?.tokens ?? 0) > 0
                  ? datum?.tokens.toLocaleString()
                  : t('No Token data'),
            },
          ],
        },
      },
      background: { fill: 'transparent' },
    } as ModelAnalyticsChartSpecs['consumption'],
    calls: {
      type: 'area',
      data: [{ id: 'model-calls', values }],
      xField: 'timeLabel',
      yField: 'calls',
      seriesField: 'model',
      axes,
      legends,
      point: { visible: false },
      background: { fill: 'transparent' },
    } as IChartSpec,
    proportion: {
      type: 'pie',
      data: [{ id: 'model-proportion', values: modelTotals }],
      valueField: 'calls',
      categoryField: 'model',
      legends,
      background: { fill: 'transparent' },
    } as IChartSpec,
    ranking: {
      type: 'bar',
      data: [
        {
          id: 'model-ranking',
          values: [...modelTotals].sort((a, b) => b.calls - a.calls),
        },
      ],
      xField: 'model',
      yField: 'calls',
      seriesField: 'model',
      legends,
      background: { fill: 'transparent' },
    } as IChartSpec,
    summary: { totalQuota, totalCalls },
  }
}
