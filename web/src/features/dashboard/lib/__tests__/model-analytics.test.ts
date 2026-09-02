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
import { describe, expect, it } from 'vitest'

import type { AnalyticsBucket } from '../../types'
import {
  buildModelAnalyticsCharts,
  createRangePreset,
  formatShanghaiRange,
  shortModelName,
} from '../model-analytics'

function bucket(bucketStart: number, quota = 0): AnalyticsBucket {
  return {
    bucket_start: bucketStart,
    bucket_end: bucketStart + 3600,
    model_name: 'doubao-seedance-2-5-260628',
    total_calls: quota > 0 ? 1 : 0,
    success_calls: quota > 0 ? 1 : 0,
    failure_calls: 0,
    running_calls: 0,
    quota,
    tokens: quota * 2,
    output_counts: {},
  }
}

describe('model analytics helpers', () => {
  it('uses Shanghai midnight for Today and keeps 24 hours rolling', () => {
    const now = new Date('2026-09-01T08:00:00Z')

    expect(createRangePreset('today', now)).toEqual({
      start: 1788192000,
      end: 1788249600,
      granularity: 'hour',
    })
    const rolling = createRangePreset('24h', now)
    expect(rolling.end - rolling.start).toBe(86_400)
  })

  it('formats Shanghai ranges and known Ark model names', () => {
    expect(
      formatShanghaiRange({ start: 1788192000, end: 1788249600 })
    ).toBe('2026-09-01 00:00 – 2026-09-01 16:00')
    expect(shortModelName('doubao-seedance-2-5-260628')).toBe('Seedance 2.5')
    expect(shortModelName('doubao-seedance-2-0-260128')).toBe('Seedance 2.0')
    expect(shortModelName('doubao-seedream-5-0-pro-260628')).toBe(
      'Seedream 5.0 Pro'
    )
    expect(shortModelName('future-model-x')).toBe('future-model-x')
  })

  it('keeps every real bucket, including empty timestamps', () => {
    const buckets = Array.from({ length: 7 }, (_, index) =>
      bucket(1788192000 + index * 3600, index < 5 ? index + 1 : 0)
    )

    const specs = buildModelAnalyticsCharts(buckets, {
      granularity: 'hour',
    })

    expect(
      specs.consumption.data[0].values.map(
        (row: { bucketStart: number }) => row.bucketStart
      )
    ).toEqual(buckets.map((row) => row.bucket_start))
  })

  it('keeps long ranges readable without changing totals', () => {
    const buckets = Array.from({ length: 24 }, (_, index) =>
      bucket(1788192000 + index * 3600, index + 1)
    )

    const specs = buildModelAnalyticsCharts(buckets, {
      granularity: 'hour',
    })

    expect('dataZoom' in specs.consumption).toBe(false)
    expect((specs.consumption as unknown as Record<string, unknown>).xField).toBe(
      'timeLabel'
    )
    expect(specs.summary.totalQuota).toBe(300)
    expect(specs.summary.totalCalls).toBe(24)
  })

  it('draws consumption with display currency values while preserving raw quota', () => {
    const specs = buildModelAnalyticsCharts([bucket(1788192000, 500000)], {
      granularity: 'hour',
    })

    expect((specs.consumption as unknown as Record<string, unknown>).yField).toBe(
      'quotaAmount'
    )
    expect(specs.consumption.data[0].values[0]).toMatchObject({
      quota: 500000,
      quotaAmount: 1,
    })
  })
})
