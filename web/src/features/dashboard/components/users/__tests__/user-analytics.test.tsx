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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import type { ModelAnalyticsData } from '../../../types'
import { UserAnalyticsSection } from '../user-analytics-section'

vi.mock('@visactor/react-vchart', () => ({
  VChart: () => <div role='img' aria-label='Chart' />,
}))

const data: ModelAnalyticsData = {
  range: {
    start: 1788192000,
    end: 1788278400,
    granularity: 'day',
    timezone: 'Asia/Shanghai',
  },
  summary: {
    total_calls: 23,
    success_calls: 23,
    failure_calls: 0,
    running_calls: 0,
    total_quota: 0,
    total_tokens: 12345,
    peak_rpm: 0,
    peak_rpm_at: 0,
    peak_tpm: 0,
    peak_tpm_at: 0,
  },
  series: [],
  models: [],
  available_models: [],
  updated_at: 1788278400,
}
const animationsDescriptor = Object.getOwnPropertyDescriptor(
  Element.prototype,
  'getAnimations'
)
let client: QueryClient
const userPage = {
  success: true,
  data: {
    items: [{ username: 'admin' }, { username: 'alice' }, { username: 'bob' }],
    total: 3,
    page: 1,
    page_size: 100,
  },
}

function renderAnalytics() {
  return render(
    <QueryClientProvider client={client}>
      <UserAnalyticsSection />
    </QueryClientProvider>
  )
}

beforeEach(() => {
  Object.defineProperty(Element.prototype, 'getAnimations', {
    configurable: true,
    value: () => [],
  })
  vi.spyOn(api, 'get').mockImplementation(async (url) => ({
    data: url === '/api/user/' ? userPage : { success: true, data },
  }))
  localStorage.clear()
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  useAuthStore
    .getState()
    .auth.setUser({ id: 1, username: 'admin', role: ROLE.ADMIN })
})

afterEach(() => {
  if (animationsDescriptor) {
    Object.defineProperty(
      Element.prototype,
      'getAnimations',
      animationsDescriptor
    )
  } else {
    Reflect.deleteProperty(Element.prototype, 'getAnimations')
  }
  client.clear()
  useAuthStore.getState().auth.reset()
  localStorage.clear()
})

describe('user analytics', () => {
  it('loads the current user on entry and rejects whitespace usernames', async () => {
    const request = vi.spyOn(api, 'get')
    const user = userEvent.setup()
    renderAnalytics()
    expect(await screen.findByTitle('23')).toBeVisible()
    expect(request).toHaveBeenCalledWith(
      '/api/data/model-analytics',
      expect.objectContaining({
        params: expect.objectContaining({ username: 'admin' }),
      })
    )
    expect(request).not.toHaveBeenCalledWith('/api/user/', expect.anything())
    await user.clear(screen.getByRole('combobox', { name: 'Username' }))
    await user.type(screen.getByRole('combobox', { name: 'Username' }), '   ')
    await user.tab()
    await user.click(screen.getByRole('button', { name: 'Search' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Enter username')
    expect(screen.getByRole('combobox', { name: 'Username' })).toHaveAttribute(
      'aria-invalid',
      'true'
    )
    expect(screen.getByText('Viewing analytics for admin')).toBeVisible()
  })

  it('queries the trimmed username and keeps it when changing or resetting time filters', async () => {
    const request = vi.mocked(api.get)
    const user = userEvent.setup()
    renderAnalytics()
    await user.clear(screen.getByRole('combobox', { name: 'Username' }))
    await user.type(
      screen.getByRole('combobox', { name: 'Username' }),
      ' alice '
    )
    await user.tab()
    await user.click(screen.getByRole('button', { name: 'Search' }))
    expect(await screen.findByTitle('23')).toBeVisible()
    expect(screen.getByText(/Viewing analytics for/)).toHaveTextContent(
      'Viewing analytics for alice'
    )
    expect(
      screen.getByRole('heading', { name: 'Consumption trend' })
    ).toBeVisible()
    expect(
      screen.getByRole('heading', { name: 'Model Call Analytics' })
    ).toBeVisible()
    expect(screen.queryByText('Current RPM')).not.toBeInTheDocument()
    expect(screen.queryByText('Current TPM')).not.toBeInTheDocument()
    expect(
      screen.queryByRole('region', { name: 'Performance health' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByText('User Consumption Ranking')
    ).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Today' }))
    await waitFor(() =>
      expect(request).toHaveBeenLastCalledWith(
        '/api/data/model-analytics',
        expect.objectContaining({
          params: expect.objectContaining({
            username: 'alice',
            granularity: 'hour',
          }),
        })
      )
    )
    await user.click(screen.getByRole('button', { name: 'Custom' }))
    expect(
      screen.queryByPlaceholderText('Filter by username')
    ).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Reset' }))
    await waitFor(() =>
      expect(request).toHaveBeenLastCalledWith(
        '/api/data/model-analytics',
        expect.objectContaining({
          params: expect.objectContaining({ username: 'alice' }),
        })
      )
    )
    expect(
      request.mock.calls.every(
        ([url]) => url === '/api/data/model-analytics' || url === '/api/user/'
      )
    ).toBe(true)
  })

  it('clears the previous user totals while the next user loads and shows an empty result', async () => {
    let resolveNext!: (value: unknown) => void
    vi.mocked(api.get).mockImplementation((url, config) => {
      if (url === '/api/user/') return Promise.resolve({ data: userPage })
      if (config?.params?.username === 'bob') {
        return new Promise((resolve) => {
          resolveNext = resolve
        })
      }
      return Promise.resolve({ data: { success: true, data } })
    })
    const user = userEvent.setup()
    renderAnalytics()
    const input = screen.getByRole('combobox', { name: 'Username' })
    expect(await screen.findByTitle('23')).toBeVisible()
    await user.clear(input)
    await user.type(input, 'bob')
    await user.tab()
    await user.click(screen.getByRole('button', { name: 'Search' }))
    await waitFor(() =>
      expect(screen.getByText(/Viewing analytics for/)).toHaveTextContent(
        'Viewing analytics for bob'
      )
    )
    expect(screen.queryByTitle('23')).not.toBeInTheDocument()
    await act(async () =>
      resolveNext({
        data: {
          success: true,
          data: {
            ...data,
            summary: { ...data.summary, total_calls: 0, total_tokens: 0 },
          },
        },
      })
    )
    expect(await screen.findAllByText('No calls')).toHaveLength(2)
  })

  it.each(['network', 'business'])(
    'recovers after a %s error when retrying the selected user',
    async (failure) => {
      const request = vi.spyOn(api, 'get')
      if (failure === 'network') {
        request.mockRejectedValueOnce(new Error('offline'))
      } else {
        request.mockResolvedValueOnce({ data: { success: false, data: null } })
      }
      request.mockResolvedValue({ data: { success: true, data } })
      const user = userEvent.setup()
      renderAnalytics()
      const retries = await screen.findAllByRole('button', { name: 'Retry' })
      expect(screen.getAllByText('Unable to load analytics')).toHaveLength(2)
      await user.click(retries[0])
      expect(await screen.findByTitle('23')).toBeVisible()
      expect(request).toHaveBeenLastCalledWith(
        '/api/data/model-analytics',
        expect.objectContaining({
          params: expect.objectContaining({ username: 'admin' }),
        })
      )
    }
  )
  it('shows all usernames on focus, filters typing, and selects with the keyboard', async () => {
    const user = userEvent.setup()
    renderAnalytics()
    const input = screen.getByRole('combobox', { name: 'Username' })
    await screen.findByTitle('23')
    await user.click(input)
    expect(await screen.findByRole('option', { name: 'alice' })).toBeVisible()
    expect(screen.getByRole('option', { name: 'bob' })).toBeVisible()
    expect(input).toHaveAttribute('aria-expanded', 'true')
    await user.clear(input)
    await user.type(input, 'ali')
    expect(screen.getByRole('option', { name: 'alice' })).toBeVisible()
    expect(
      screen.queryByRole('option', { name: 'bob' })
    ).not.toBeInTheDocument()
    await user.keyboard('{ArrowDown}{Enter}')
    expect(input).toHaveValue('alice')
    await user.click(screen.getByRole('button', { name: 'Search' }))
    expect(screen.getByText('Viewing analytics for alice')).toBeVisible()
    await user.click(input)
    expect(await screen.findByRole('option', { name: 'bob' })).toBeVisible()
    await user.clear(input)
    await user.type(input, 'missing')
    expect(await screen.findByText('No results found')).toBeVisible()
    await user.keyboard('{Escape}')
    expect(input).toHaveAttribute('aria-expanded', 'false')
  })

  it('loads subsequent user pages and supports pointer selection', async () => {
    vi.mocked(api.get).mockImplementation(async (url, config) => {
      if (url !== '/api/user/') return { data: { success: true, data } }
      return {
        data: {
          success: true,
          data: {
            items: [
              { username: config?.params?.p === 1 ? 'admin' : 'last-user' },
            ],
            total: 2,
            page: config?.params?.p,
            page_size: 1,
          },
        },
      }
    })
    const user = userEvent.setup()
    renderAnalytics()
    await user.click(screen.getByRole('combobox', { name: 'Username' }))
    await user.click(await screen.findByRole('option', { name: 'last-user' }))
    await user.click(screen.getByRole('button', { name: 'Search' }))
    expect(screen.getByText('Viewing analytics for last-user')).toBeVisible()
  })

  it('retries a failed user list without losing the current statistics', async () => {
    let fail = true
    vi.mocked(api.get).mockImplementation(async (url) => {
      if (url !== '/api/user/') return { data: { success: true, data } }
      return { data: fail ? { success: false } : userPage }
    })
    const user = userEvent.setup()
    renderAnalytics()
    await screen.findByTitle('23')
    await user.click(screen.getByRole('combobox', { name: 'Username' }))
    expect(await screen.findByText('Failed to load users')).toBeVisible()
    expect(screen.getByTitle('23')).toBeVisible()
    fail = false
    await user.click(screen.getByRole('button', { name: 'Retry' }))
    expect(await screen.findByRole('option', { name: 'alice' })).toBeVisible()
  })
})
