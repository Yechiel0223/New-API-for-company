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
// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import type { ModelHealthData } from '../../../types'
import { PerformanceOverview } from '../performance-overview'

const health: ModelHealthData = {
  window_hours: 24,
  overall: {
    status: 'warning',
    total_calls: 4,
    success_calls: 3,
    failure_calls: 1,
    running_calls: 0,
    stuck_calls: 0,
    success_rate: 75,
  },
  current_load: { rpm: 0.8, tpm: 2500, window_minutes: 5 },
  models: [
    {
      model_name: 'doubao-seedream-5-0-pro-260628',
      status: 'warning',
      total_calls: 4,
      success_calls: 3,
      failure_calls: 1,
      running_calls: 0,
      stuck_calls: 0,
      success_rate: 75,
      p50_ms: 50_000,
      p95_ms: 55_000,
      successful_duration_samples: 3,
      latest_failure_at: 1788249000,
      latest_failure_reason: 'upstream timeout',
    },
  ],
  updated_at: 1788249600,
}

describe('PerformanceOverview', () => {
  it('renders direct per-model health facts without legacy throughput', () => {
    render(
      <PerformanceOverview
        data={health}
        loading={false}
        error={false}
        onWindowChange={vi.fn()}
        onRetry={vi.fn()}
        onModelClick={vi.fn()}
      />
    )

    expect(screen.getByText('Seedream 5.0 Pro')).toBeVisible()
    expect(screen.getByText('Success 3/4 · 75%')).toBeVisible()
    expect(screen.getByText('P50 50s')).toBeVisible()
    expect(screen.getByText('P95 55s')).toBeVisible()
    expect(screen.queryByText('Throughput')).not.toBeInTheDocument()
  })

  it('shows the no-calls status and last update for an empty window', () => {
    render(
      <PerformanceOverview
        data={{
          ...health,
          overall: {
            status: 'no_data',
            total_calls: 0,
            success_calls: 0,
            failure_calls: 0,
            running_calls: 0,
            stuck_calls: 0,
            success_rate: 0,
          },
          models: [],
        }}
        loading={false}
        error={false}
        onWindowChange={vi.fn()}
        onRetry={vi.fn()}
        onModelClick={vi.fn()}
      />
    )

    expect(screen.getByText('No calls')).toBeVisible()
    expect(screen.getByText(/Last updated:/)).toBeVisible()
  })
})
