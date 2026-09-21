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
import {
  useReactTable,
  getCoreRowModel,
  flexRender,
} from '@tanstack/react-table'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import type { ApiKey } from '../../types'
import { useApiKeysColumns } from '../api-keys-columns'
import { ApiKeysMutateDrawer } from '../api-keys-mutate-drawer'
import { ApiKeysProvider } from '../api-keys-provider'

const key: ApiKey = {
  id: 7,
  name: 'restricted',
  key: '',
  status: 1,
  remain_quota: 500000,
  used_quota: 0,
  unlimited_quota: false,
  expired_time: 123,
  created_time: 1,
  accessed_time: 0,
  group: 'vip',
  auto_groups: ['vip'],
  cross_group_retry: true,
  model_limits_enabled: true,
  model_limits: 'gpt-4',
  allow_ips: '127.0.0.1',
}
let client: QueryClient
beforeEach(() => {
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/token/7') return { data: { success: true, data: key } }
    throw new Error(`Unexpected GET ${url}`)
  })
  vi.spyOn(api, 'post').mockResolvedValue({ data: { success: true } })
  vi.spyOn(api, 'put').mockResolvedValue({ data: { success: true } })
})
afterEach(() => {
  client.clear()
  localStorage.clear()
})
function renderEditor(currentRow?: ApiKey) {
  const onOpenChange = vi.fn()
  render(
    <QueryClientProvider client={client}>
      <ApiKeysProvider>
        <ApiKeysMutateDrawer
          open
          currentRow={currentRow}
          onOpenChange={onOpenChange}
        />
      </ApiKeysProvider>
    </QueryClientProvider>
  )
  return onOpenChange
}
function KeyHeaders() {
  const table = useReactTable({
    data: [],
    columns: useApiKeysColumns(0),
    getCoreRowModel: getCoreRowModel(),
  })
  return (
    <table>
      <thead>
        {table.getHeaderGroups().map((group) => (
          <tr key={group.id}>
            {group.headers.map((header) => (
              <th key={header.id}>
                {flexRender(
                  header.column.columnDef.header,
                  header.getContext()
                )}
              </th>
            ))}
          </tr>
        ))}
      </thead>
    </table>
  )
}
describe('simplified API key editor', () => {
  it('removes group and access restriction columns from the key list', () => {
    render(<KeyHeaders />)
    expect(screen.getByRole('columnheader', { name: 'Name' })).toBeVisible()
    for (const name of ['Group', 'Models', 'IP Restriction', 'Expires']) {
      expect(
        screen.queryByRole('columnheader', { name })
      ).not.toBeInTheDocument()
    }
  })
  it('hides restrictions and creates each key with the default configuration', async () => {
    const user = userEvent.setup()
    renderEditor()
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Save changes' })).toBeEnabled()
    )
    for (const label of [
      'Group',
      'Expiration Time',
      'Model Limits',
      'IP Whitelist (supports CIDR)',
    ]) {
      expect(screen.queryByLabelText(label)).not.toBeInTheDocument()
    }
    await user.type(screen.getByLabelText('Name'), 'batch')
    fireEvent.change(screen.getByLabelText('Quantity'), {
      target: { value: '2' },
    })
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(api.post).toHaveBeenCalledTimes(2))
    for (const [, payload] of vi.mocked(api.post).mock.calls) {
      expect(payload).toMatchObject({
        group: 'default',
        auto_groups: [],
        cross_group_retry: false,
        expired_time: -1,
        model_limits_enabled: false,
        model_limits: '',
        allow_ips: '',
      })
    }
    expect(api.get).not.toHaveBeenCalled()
  })
  it('saves old restricted keys with defaults only after Save changes', async () => {
    const user = userEvent.setup()
    const close = renderEditor(key)
    await waitFor(() =>
      expect(screen.getByLabelText('Name')).toHaveValue('restricted')
    )
    expect(api.put).not.toHaveBeenCalled()
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(close).toHaveBeenCalledWith(false))
    expect(api.put).toHaveBeenCalledWith(
      '/api/token/',
      expect.objectContaining({
        id: 7,
        group: 'default',
        expired_time: -1,
        model_limits_enabled: false,
        allow_ips: '',
        remain_quota: 500000,
      })
    )
  })
  it('prevents saving when the existing key cannot be loaded', async () => {
    vi.mocked(api.get).mockRejectedValue(new Error('offline'))
    renderEditor(key)
    await waitFor(() => expect(api.get).toHaveBeenCalled())
    expect(screen.getByRole('button', { name: 'Save changes' })).toBeDisabled()
    expect(api.put).not.toHaveBeenCalled()
  })
})
