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
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
} from '@/stores/system-config-store'

import { UserQuotaDialog } from '../user-quota-dialog'

const animations = Object.getOwnPropertyDescriptor(
  Element.prototype,
  'getAnimations'
)
beforeEach(() => {
  Object.defineProperty(Element.prototype, 'getAnimations', {
    configurable: true,
    value: () => [],
  })
  localStorage.clear()
})
afterEach(() => {
  if (animations) {
    Object.defineProperty(Element.prototype, 'getAnimations', animations)
  } else Reflect.deleteProperty(Element.prototype, 'getAnimations')
  localStorage.clear()
})

beforeEach(() => {
  useSystemConfigStore
    .getState()
    .setConfig({ currency: { ...DEFAULT_CURRENCY_CONFIG } })
})
afterEach(() => {
  useSystemConfigStore
    .getState()
    .setConfig({ currency: { ...DEFAULT_CURRENCY_CONFIG } })
})

function renderQuota() {
  const props = {
    open: true,
    userId: 7,
    currentQuota: 1000000,
    onOpenChange: vi.fn(),
    onSuccess: vi.fn(),
  }
  return { props, ...render(<UserQuotaDialog {...props} />) }
}

describe('quota replacement', () => {
  it('prefills the balance and replaces it with the entered amount', async () => {
    const request = vi
      .spyOn(api, 'post')
      .mockResolvedValue({ data: { success: true } })
    const user = userEvent.setup()
    const { props } = renderQuota()
    const input = screen.getByRole('spinbutton', { name: /Remaining Quota/ })
    expect(input).toHaveValue(2)
    expect(screen.queryByText('Mode')).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Add' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Subtract' })
    ).not.toBeInTheDocument()
    await user.clear(input)
    await user.type(input, '3.25')
    await user.click(screen.getByRole('button', { name: 'Confirm' }))
    await waitFor(() => expect(props.onSuccess).toHaveBeenCalledOnce())
    expect(request).toHaveBeenCalledExactlyOnceWith('/api/user/manage', {
      id: 7,
      action: 'add_quota',
      mode: 'override',
      value: 1625000,
    })
    expect(props.onOpenChange).toHaveBeenCalledWith(false)
  })

  it('allows explicitly setting the balance to zero', async () => {
    const request = vi
      .spyOn(api, 'post')
      .mockResolvedValue({ data: { success: true } })
    const user = userEvent.setup()
    renderQuota()
    const input = screen.getByRole('spinbutton')
    await user.clear(input)
    await user.type(input, '0{Enter}')
    await waitFor(() =>
      expect(request).toHaveBeenCalledExactlyOnceWith('/api/user/manage', {
        id: 7,
        action: 'add_quota',
        mode: 'override',
        value: 0,
      })
    )
  })

  it.each(['', '-1', '99999999999999999'])(
    'rejects invalid or unsafe quota %j without a request',
    async (amount) => {
      const request = vi.spyOn(api, 'post')
      const user = userEvent.setup()
      renderQuota()
      const input = screen.getByRole('spinbutton')
      await user.clear(input)
      if (amount) await user.type(input, amount)
      expect(input).toHaveAttribute('aria-invalid', 'true')
      expect(screen.getByRole('alert')).toBeVisible()
      expect(screen.getByRole('button', { name: 'Confirm' })).toBeDisabled()
      await user.keyboard('{Enter}')
      expect(request).not.toHaveBeenCalled()
    }
  )

  it('discards an unsaved amount when reopened with a refreshed balance', async () => {
    const user = userEvent.setup()
    const { props, rerender } = renderQuota()
    await user.clear(screen.getByRole('spinbutton'))
    await user.type(screen.getByRole('spinbutton'), '99')
    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    rerender(<UserQuotaDialog {...props} open={false} />)
    rerender(<UserQuotaDialog {...props} currentQuota={500000} />)
    expect(screen.getByRole('spinbutton')).toHaveValue(1)
  })

  it.each(['network', 'business'])(
    'preserves the amount after a %s failure and allows retry',
    async (failure) => {
      const request = vi.spyOn(api, 'post')
      if (failure === 'network') {
        request.mockRejectedValueOnce(new Error('offline'))
      } else request.mockResolvedValueOnce({ data: { success: false } })
      request.mockResolvedValue({ data: { success: true } })
      const user = userEvent.setup()
      const { props } = renderQuota()
      await user.click(screen.getByRole('button', { name: 'Confirm' }))
      await waitFor(() =>
        expect(screen.getByRole('button', { name: 'Confirm' })).toBeEnabled()
      )
      expect(props.onSuccess).not.toHaveBeenCalled()
      expect(screen.getByRole('spinbutton')).toHaveValue(2)
      await user.click(screen.getByRole('button', { name: 'Confirm' }))
      await waitFor(() => expect(props.onSuccess).toHaveBeenCalledOnce())
    }
  )

  it.each([
    { type: 'CNY' as const, rate: 7, input: '14', expected: 1000000 },
    { type: 'TOKENS' as const, rate: 1, input: '123', expected: 123 },
  ])(
    'converts the configured $type display before replacing quota',
    async ({ type, rate, input, expected }) => {
      useSystemConfigStore.getState().setConfig({
        currency: {
          ...DEFAULT_CURRENCY_CONFIG,
          quotaDisplayType: type,
          usdExchangeRate: rate,
        },
      })
      const request = vi
        .spyOn(api, 'post')
        .mockResolvedValue({ data: { success: true } })
      const user = userEvent.setup()
      renderQuota()
      await user.clear(screen.getByRole('spinbutton'))
      await user.type(screen.getByRole('spinbutton'), input)
      await user.click(screen.getByRole('button', { name: 'Confirm' }))
      expect(request).toHaveBeenCalledWith('/api/user/manage', {
        id: 7,
        action: 'add_quota',
        mode: 'override',
        value: expected,
      })
    }
  )
})
