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
import {
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import type { ApiKey } from '../../types'
import { useApiKeysColumns } from '../api-keys-columns'
import { ApiKeysDialogs } from '../api-keys-dialogs'
import { ApiKeysProvider } from '../api-keys-provider'

const key: ApiKey = {
  id: 7,
  name: 'legacy-demo',
  key: 'masked',
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

function KeyTable(props: { status: number; filter?: string[] }) {
  const table = useReactTable({
    data: [{ ...key, status: props.status }],
    columns: useApiKeysColumns(0),
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    state: {
      columnFilters: props.filter
        ? [{ id: 'status', value: props.filter }]
        : [],
    },
  })
  return (
    <>
      <table>
        <tbody>
          {table.getRowModel().rows.map((row) => (
            <tr key={row.id}>
              {row.getVisibleCells().map((cell) => (
                <td key={cell.id}>
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
      <ApiKeysDialogs />
    </>
  )
}

function renderKey(status = 1, filter?: string[]) {
  render(
    <ApiKeysProvider>
      <KeyTable status={status} filter={filter} />
    </ApiKeysProvider>
  )
}

beforeEach(() => {
  vi.spyOn(api, 'get').mockRejectedValue(new Error('Unexpected GET'))
  vi.spyOn(api, 'put').mockResolvedValue({ data: { success: true } })
  vi.spyOn(api, 'delete').mockResolvedValue({ data: { success: true } })
  vi.spyOn(api, 'post').mockResolvedValue({
    data: { success: true, data: { key: 'test-only-secret' } },
  })
})

describe('simplified key actions', () => {
  it('only exposes copy, disable and delete without changing legacy data on load', () => {
    renderKey()
    expect(
      screen
        .getAllByRole('button')
        .map((button) => button.getAttribute('aria-label'))
    ).toEqual(['Copy API key', 'Disable', 'Delete'])
    expect(api.get).not.toHaveBeenCalled()
    expect(api.put).not.toHaveBeenCalled()
    expect(api.post).not.toHaveBeenCalled()
  })
  it('copies the resolved key instead of the masked value', async () => {
    const user = userEvent.setup()
    const copy = vi.spyOn(navigator.clipboard, 'writeText').mockResolvedValue()
    renderKey()
    await user.click(screen.getByRole('button', { name: 'Copy API key' }))
    await waitFor(() =>
      expect(copy).toHaveBeenCalledWith('sk-test-only-secret')
    )
    expect(api.post).toHaveBeenCalledWith('/api/token/7/key')
  })
  it('disables a key through the status-only API without clearing its old restrictions', async () => {
    const user = userEvent.setup()
    renderKey()
    await user.click(screen.getByRole('button', { name: 'Disable' }))
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith('/api/token/?status_only=true', {
        id: 7,
        status: 2,
      })
    )
  })
  it('allows retrying when disabling fails', async () => {
    const user = userEvent.setup()
    vi.mocked(api.put).mockRejectedValue(new Error('offline'))
    renderKey()
    await user.click(screen.getByRole('button', { name: 'Disable' }))
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Disable' })).toBeEnabled()
    )
    expect(screen.getByText('Enabled')).toBeVisible()
  })
  it.each([2, 3, 4])(
    'shows legacy status %s as disabled and keeps it in the disabled filter',
    (status) => {
      renderKey(status, ['2'])
      expect(screen.getByText('Disabled')).toBeVisible()
      expect(screen.getByRole('button', { name: 'Disable' })).toBeDisabled()
      expect(screen.queryByText('Expired')).not.toBeInTheDocument()
      expect(screen.queryByText('Exhausted')).not.toBeInTheDocument()
      expect(
        screen.queryByRole('button', { name: 'Enable' })
      ).not.toBeInTheDocument()
    }
  )
  it('requires confirmation before deleting a key', async () => {
    const user = userEvent.setup()
    renderKey()
    await user.click(screen.getByRole('button', { name: 'Delete' }))
    const dialog = await screen.findByRole('alertdialog')
    expect(api.delete).not.toHaveBeenCalled()
    await user.click(within(dialog).getByRole('button', { name: 'Delete' }))
    await waitFor(() =>
      expect(api.delete).toHaveBeenCalledWith('/api/token/7/')
    )
  })
})
