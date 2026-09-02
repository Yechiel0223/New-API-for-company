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
import { describe, expect, it, vi } from 'vitest'

import type { DashboardFilters } from '../../../types'
import { ModelAnalyticsToolbar } from '../model-analytics-toolbar'

const filters: DashboardFilters = {
  start_timestamp: new Date('2026-08-26T06:00:00Z'),
  end_timestamp: new Date('2026-09-02T06:00:00Z'),
  time_granularity: 'day',
  username: '',
  models: [],
}

describe('ModelAnalyticsToolbar', () => {
  it('shows the active range and applies the seven-day aggregation', async () => {
    const user = userEvent.setup()
    const onFiltersChange = vi.fn()
    render(
      <ModelAnalyticsToolbar
        filters={filters}
        availableModels={[]}
        onFiltersChange={onFiltersChange}
        onRefresh={vi.fn()}
        refreshing={false}
      />
    )

    expect(screen.getByText(/Current range:/i)).toBeVisible()
    const sevenDays = screen.getByRole('button', { name: /7 Days/i })
    await user.click(sevenDays)

    expect(onFiltersChange).toHaveBeenCalledWith(
      expect.objectContaining({ time_granularity: 'day' })
    )
    expect(sevenDays).toHaveAttribute('aria-pressed', 'true')
  })

  it('supports Today, 24 hours, 30 days, and manual refresh', async () => {
    const user = userEvent.setup()
    const onFiltersChange = vi.fn()
    const onRefresh = vi.fn()
    render(
      <ModelAnalyticsToolbar
        filters={filters}
        availableModels={[]}
        onFiltersChange={onFiltersChange}
        onRefresh={onRefresh}
        refreshing={false}
      />
    )

    await user.click(screen.getByRole('button', { name: /Today/i }))
    expect(onFiltersChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ time_granularity: 'hour' })
    )
    await user.click(screen.getByRole('button', { name: /24 Hours/i }))
    expect(onFiltersChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ time_granularity: 'hour' })
    )
    await user.click(screen.getByRole('button', { name: /30 Days/i }))
    expect(onFiltersChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ time_granularity: 'day' })
    )
    await user.click(screen.getByRole('button', { name: /Refresh/i }))
    expect(onRefresh).toHaveBeenCalledTimes(1)
  })
})
