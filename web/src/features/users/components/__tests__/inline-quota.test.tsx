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
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'
import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
} from '@/stores/system-config-store'

import type { User } from '../../types'
import { UsersMutateDrawer } from '../users-mutate-drawer'
import { UsersProvider } from '../users-provider'

const target: User = {
  id: 7,
  username: 'alice',
  display_name: 'Alice',
  role: 1,
  status: 1,
  quota: 1000000,
  used_quota: 0,
  request_count: 0,
  group: 'default',
}
let client: QueryClient
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
  useSystemConfigStore
    .getState()
    .setConfig({ currency: { ...DEFAULT_CURRENCY_CONFIG } })
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  useAuthStore
    .getState()
    .auth.setUser({ id: 1, username: 'admin', role: ROLE.SUPER_ADMIN })
  vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/user/7') return { data: { success: true, data: target } }
    if (url === '/api/group/') {
      return { data: { success: true, data: ['default'] } }
    }
    if (url === '/api/authz/catalog') {
      return { data: { data: { resources: [], roles: [] } } }
    }
    throw new Error(`Unexpected request: ${url}`)
  })
  vi.spyOn(api, 'put').mockResolvedValue({ data: { success: true } })
  vi.spyOn(api, 'post').mockResolvedValue({ data: { success: true } })
})
afterEach(() => {
  client.clear()
  useAuthStore.getState().auth.reset()
  useSystemConfigStore
    .getState()
    .setConfig({ currency: { ...DEFAULT_CURRENCY_CONFIG } })
  if (animations) {
    Object.defineProperty(Element.prototype, 'getAnimations', animations)
  } else Reflect.deleteProperty(Element.prototype, 'getAnimations')
  localStorage.clear()
})

async function renderEditor() {
  const onOpenChange = vi.fn()
  render(
    <QueryClientProvider client={client}>
      <UsersProvider>
        <UsersMutateDrawer
          open
          currentRow={target}
          onOpenChange={onOpenChange}
        />
      </UsersProvider>
    </QueryClientProvider>
  )
  const input = screen.getByRole('spinbutton', { name: /Remaining Quota/ })
  await waitFor(() => expect(input).toBeEnabled())
  return { input, onOpenChange }
}

describe('inline quota editing', () => {
  it.each([
    { inputValue: '3.25', expected: 1625000 },
    { inputValue: '0', expected: 0 },
  ])(
    'saves the edited $inputValue balance only after Save changes',
    async ({ inputValue, expected }) => {
      const user = userEvent.setup()
      const { input, onOpenChange } = await renderEditor()
      expect(input).toHaveValue(2)
      expect(
        screen.queryByRole('button', { name: 'Adjust Quota' })
      ).not.toBeInTheDocument()
      await user.clear(input)
      await user.type(input, inputValue)
      await user.tab()
      expect(input).toHaveValue(Number(inputValue))
      expect(api.post).not.toHaveBeenCalled()
      await user.click(screen.getByRole('button', { name: 'Save changes' }))
      await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false))
      expect(api.post).toHaveBeenCalledExactlyOnceWith('/api/user/manage', {
        id: 7,
        action: 'add_quota',
        mode: 'override',
        value: expected,
      })
      expect(api.put).not.toHaveBeenCalled()
    }
  )

  it('does not overwrite an unchanged quota while saving other user details', async () => {
    const user = userEvent.setup()
    await renderEditor()
    await user.type(screen.getByLabelText('Password'), 'Updated123!')
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(api.put).toHaveBeenCalledOnce())
    expect(api.post).not.toHaveBeenCalled()
  })

  it('does not write quota when closing an unsaved edit', async () => {
    const user = userEvent.setup()
    const { input, onOpenChange } = await renderEditor()
    await user.clear(input)
    await user.type(input, '5')
    await user.click(screen.getAllByRole('button', { name: 'Close' })[0])
    expect(onOpenChange).toHaveBeenCalledWith(false)
    expect(api.post).not.toHaveBeenCalled()
  })

  it.each(['', '-1', '99999999999999999'])(
    'rejects invalid quota %j before either update',
    async (amount) => {
      const user = userEvent.setup()
      const { input, onOpenChange } = await renderEditor()
      await user.clear(input)
      if (amount) await user.type(input, amount)
      if (amount) expect(input).toHaveValue(Number(amount))
      await user.click(screen.getByRole('button', { name: 'Save changes' }))
      expect(
        await screen.findByText('Enter a valid non-negative quota amount')
      ).toBeVisible()
      expect(input).toHaveAttribute('aria-invalid', 'true')
      expect(api.post).not.toHaveBeenCalled()
      expect(api.put).not.toHaveBeenCalled()
      expect(onOpenChange).not.toHaveBeenCalled()
    }
  )

  it.each(['network', 'business'])(
    'retains the quota after a %s failure and retries without repeating saved user details',
    async (failure) => {
      const request = vi.mocked(api.post)
      if (failure === 'network') {
        request.mockRejectedValueOnce(new Error('offline'))
      } else request.mockResolvedValueOnce({ data: { success: false } })
      const user = userEvent.setup()
      const { input, onOpenChange } = await renderEditor()
      await user.type(screen.getByLabelText('Password'), 'Updated123!')
      await user.clear(input)
      await user.type(input, '7')
      await user.click(screen.getByRole('button', { name: 'Save changes' }))
      expect(
        await screen.findByText(
          'User details saved, but quota update failed. Save again to retry the quota update.'
        )
      ).toBeVisible()
      expect(input).toHaveValue(7)
      expect(onOpenChange).not.toHaveBeenCalled()
      await user.click(screen.getByRole('button', { name: 'Save changes' }))
      await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false))
      expect(api.put).toHaveBeenCalledOnce()
      expect(api.post).toHaveBeenCalledTimes(2)
    }
  )

  it('does not update quota when saving the user details fails', async () => {
    vi.mocked(api.put).mockResolvedValue({ data: { success: false } })
    const user = userEvent.setup()
    const { input, onOpenChange } = await renderEditor()
    await user.type(screen.getByLabelText('Password'), 'Updated123!')
    await user.clear(input)
    await user.type(input, '7')
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(api.put).toHaveBeenCalledOnce())
    expect(api.post).not.toHaveBeenCalled()
    expect(onOpenChange).not.toHaveBeenCalled()
  })
})
