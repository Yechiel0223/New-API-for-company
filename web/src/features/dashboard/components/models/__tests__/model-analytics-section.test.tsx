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
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { useSystemConfigStore } from '@/stores/system-config-store'

import type { ModelAnalyticsData } from '../../../types'
import { LogStatCards } from '../log-stat-cards'

const analytics: ModelAnalyticsData = {
  range: {
    start: 1788192000,
    end: 1788278400,
    granularity: 'hour',
    timezone: 'Asia/Shanghai',
  },
  summary: {
    total_calls: 20,
    success_calls: 19,
    failure_calls: 1,
    running_calls: 0,
    total_quota: 18_084_285,
    total_tokens: 4_226_311,
    peak_rpm: 3.2,
    peak_rpm_at: 1788249600,
    peak_tpm: 100_000,
    peak_tpm_at: 1788249600,
  },
  series: [],
  models: [],
  available_models: [],
  updated_at: 1788249600,
}

describe('LogStatCards', () => {
  beforeEach(() => {
    useSystemConfigStore.getState().setConfig({
      currency: {
        displayInCurrency: true,
        quotaDisplayType: 'CNY',
        quotaPerUnit: 500_000,
        usdExchangeRate: 7,
        customCurrencySymbol: '¤',
        customCurrencyExchangeRate: 1,
      },
    })
  })

  it('renders selected totals without RPM or TPM', () => {
    render(
      <LogStatCards
        analytics={analytics}
        loading={false}
        error={false}
        onRetry={vi.fn()}
      />
    )

    expect(screen.getByText('20')).toBeVisible()
    expect(screen.getByText('¥253.18')).toBeVisible()
    expect(screen.getByText('4,226,311')).toBeVisible()
    expect(screen.queryByText('Current RPM')).not.toBeInTheDocument()
    expect(screen.queryByText('Current TPM')).not.toBeInTheDocument()
  })

  it('shows explicit retry state after an API error', async () => {
    const user = userEvent.setup()
    const onRetry = vi.fn()
    render(
      <LogStatCards
        analytics={undefined}
        loading={false}
        error
        onRetry={onRetry}
      />
    )

    expect(screen.getAllByText('--')).toHaveLength(3)
    await user.click(screen.getByRole('button', { name: /Retry/i }))
    expect(onRetry).toHaveBeenCalledTimes(1)
  })
})
