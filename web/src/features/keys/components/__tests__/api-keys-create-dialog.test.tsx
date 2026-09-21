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
  useReactTable,
  getCoreRowModel,
  flexRender,
} from '@tanstack/react-table'
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { useApiKeysColumns } from '../api-keys-columns'
import { ApiKeysCreateDialog } from '../api-keys-create-dialog'
import { ApiKeysProvider } from '../api-keys-provider'

beforeEach(() => {
  vi.spyOn(api, 'post').mockResolvedValue({ data: { success: true } })
})

function CreateDialog(props: { onClose: () => void }) {
  const [open, setOpen] = useState(true)
  return (
    <ApiKeysProvider>
      <button type='button' onClick={() => setOpen(true)}>
        Open
      </button>
      <ApiKeysCreateDialog
        open={open}
        onOpenChange={(value) => {
          setOpen(value)
          if (!value) props.onClose()
        }}
      />
    </ApiKeysProvider>
  )
}

function renderCreator() {
  const onClose = vi.fn()
  render(<CreateDialog onClose={onClose} />)
  return onClose
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

describe('single-key creation dialog', () => {
  it('opens a centered dialog with only a focused name field', async () => {
    renderCreator()
    const dialog = screen.getByRole('dialog', { name: 'Create API Key' })
    expect(dialog).toHaveClass('top-1/2', 'left-1/2', 'sm:max-w-sm')
    expect(screen.getAllByRole('textbox')).toHaveLength(1)
    await waitFor(() => expect(screen.getByLabelText('Name')).toHaveFocus())
    expect(screen.queryByLabelText('Quantity')).not.toBeInTheDocument()
    expect(screen.queryByRole('switch')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Confirm' })).toBeDisabled()
    expect(api.post).not.toHaveBeenCalled()
  })
  it.each(['click', 'enter'])(
    'creates exactly one unrestricted key using %s',
    async (mode) => {
      const user = userEvent.setup()
      const close = renderCreator()
      await user.type(screen.getByLabelText('Name'), '  example  ')
      if (mode === 'enter') await user.keyboard('{Enter}')
      else await user.click(screen.getByRole('button', { name: 'Confirm' }))
      await waitFor(() => expect(close).toHaveBeenCalledTimes(1))
      expect(api.post).toHaveBeenCalledExactlyOnceWith('/api/token/', {
        name: 'example',
        unlimited_quota: true,
        remain_quota: 0,
        group: 'default',
        auto_groups: [],
        cross_group_retry: false,
        expired_time: -1,
        model_limits_enabled: false,
        model_limits: '',
        allow_ips: '',
      })
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    }
  )
  it.each(['network', 'server'])(
    'retains the name after a %s failure and allows retry',
    async (mode) => {
      const user = userEvent.setup()
      if (mode === 'network') {
        vi.mocked(api.post).mockRejectedValueOnce(new Error('offline'))
      } else {
        vi.mocked(api.post).mockResolvedValueOnce({ data: { success: false } })
      }
      const close = renderCreator()
      await user.type(screen.getByLabelText('Name'), 'retry')
      await user.click(screen.getByRole('button', { name: 'Confirm' }))
      await waitFor(() => expect(api.post).toHaveBeenCalledTimes(1))
      expect(close).not.toHaveBeenCalled()
      expect(screen.getByLabelText('Name')).toHaveValue('retry')
      await user.click(screen.getByRole('button', { name: 'Confirm' }))
      await waitFor(() => expect(close).toHaveBeenCalledTimes(1))
    }
  )
  it('rejects whitespace on keyboard submission without sending a request', async () => {
    const user = userEvent.setup()
    renderCreator()
    await user.type(screen.getByLabelText('Name'), '   ')
    expect(screen.getByRole('button', { name: 'Confirm' })).toBeDisabled()
    await user.keyboard('{Enter}')
    expect(api.post).not.toHaveBeenCalled()
  })
  it.each(['Cancel', 'Escape'])(
    'clears the name after closing with %s and reopening',
    async (mode) => {
      const user = userEvent.setup()
      const close = renderCreator()
      await user.type(screen.getByLabelText('Name'), 'discard')
      if (mode === 'Cancel') {
        await user.click(screen.getByRole('button', { name: 'Cancel' }))
      } else {
        await user.keyboard('{Escape}')
      }
      await waitFor(() => expect(close).toHaveBeenCalledTimes(1))
      await user.click(screen.getByRole('button', { name: 'Open' }))
      expect(screen.getByLabelText('Name')).toHaveValue('')
      expect(api.post).not.toHaveBeenCalled()
    }
  )
  it('blocks duplicate submission and dismissal while the request is pending', async () => {
    const user = userEvent.setup()
    let resolve!: (value: { data: { success: boolean } }) => void
    vi.mocked(api.post).mockImplementationOnce(
      () =>
        new Promise((done) => {
          resolve = done
        })
    )
    const close = renderCreator()
    await user.type(screen.getByLabelText('Name'), 'once')
    await user.click(screen.getByRole('button', { name: 'Confirm' }))
    await waitFor(() => expect(api.post).toHaveBeenCalledTimes(1))
    expect(screen.getByRole('button', { name: 'Creating...' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeDisabled()
    expect(screen.getByLabelText('Name')).toBeDisabled()
    await user.keyboard('{Escape}')
    expect(close).not.toHaveBeenCalled()
    const form = screen.getByLabelText('Name').closest('form')
    if (!form) throw new Error('Missing creation form')
    fireEvent.submit(form)
    await act(async () => resolve({ data: { success: true } }))
    expect(api.post).toHaveBeenCalledTimes(1)
  })
  it('keeps removed columns out of the key list', () => {
    render(<KeyHeaders />)
    for (const name of [
      'Group',
      'Models',
      'IP Restriction',
      'Expires',
      'Quota',
    ]) {
      expect(
        screen.queryByRole('columnheader', { name })
      ).not.toBeInTheDocument()
    }
  })
})
