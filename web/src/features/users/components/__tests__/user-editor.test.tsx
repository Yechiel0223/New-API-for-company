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
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import type { User } from '../../types'
import { useUsersColumns } from '../users-columns'
import { UsersMutateDrawer } from '../users-mutate-drawer'
import { UsersProvider } from '../users-provider'

const target: User = {
  id: 7,
  username: 'alice',
  display_name: 'Legacy display',
  remark: 'Existing note',
  role: 1,
  status: 1,
  quota: 500000,
  used_quota: 0,
  request_count: 0,
  group: 'default',
  github_id: 'github-identity',
  wechat_id: 'wechat-identity',
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
})
afterEach(() => {
  if (animations) {
    Object.defineProperty(Element.prototype, 'getAnimations', animations)
  } else Reflect.deleteProperty(Element.prototype, 'getAnimations')
  localStorage.clear()
})

beforeEach(() => {
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
})
afterEach(() => {
  client.clear()
  useAuthStore.getState().auth.reset()
})

function renderEditor(currentRow?: User) {
  return render(
    <QueryClientProvider client={client}>
      <UsersProvider>
        <UsersMutateDrawer
          open
          onOpenChange={vi.fn()}
          currentRow={currentRow}
        />
      </UsersProvider>
    </QueryClientProvider>
  )
}

function UsernameColumn() {
  const columns = useUsersColumns().filter(
    (column) => 'accessorKey' in column && column.accessorKey === 'username'
  )
  const table = useReactTable({
    data: [target],
    columns,
    getCoreRowModel: getCoreRowModel(),
  })
  return (
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
  )
}

describe('simplified user editor', () => {
  it('hides removed fields while retaining existing metadata in updates', async () => {
    const request = vi
      .spyOn(api, 'put')
      .mockResolvedValue({ data: { success: true } })
    const user = userEvent.setup()
    renderEditor(target)
    await waitFor(() =>
      expect(screen.getByRole('textbox', { name: 'Username' })).toHaveValue(
        'alice'
      )
    )
    expect(screen.queryByLabelText('Display Name')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Remark')).not.toBeInTheDocument()
    expect(screen.queryByText('Binding Information')).not.toBeInTheDocument()
    expect(screen.queryByText('GitHub')).not.toBeInTheDocument()
    expect(screen.queryByText('WeChat')).not.toBeInTheDocument()
    await user.type(
      screen.getByLabelText('Password', { exact: true }),
      'NewPass123!'
    )
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() =>
      expect(request).toHaveBeenCalledWith(
        '/api/user/',
        expect.objectContaining({
          id: 7,
          username: 'alice',
          display_name: 'Legacy display',
          remark: 'Existing note',
          password: 'NewPass123!',
        })
      )
    )
  })

  it('blocks balance edits and saving until current user details finish loading', async () => {
    let resolveUser!: (value: unknown) => void
    const original = vi.mocked(api.get).getMockImplementation()
    if (!original) throw new Error('Missing network fixture')
    vi.mocked(api.get).mockImplementation((url, config) => {
      if (url === '/api/user/7') {
        return new Promise((resolve) => {
          resolveUser = resolve
        })
      }
      return original(url, config)
    })
    renderEditor(target)
    expect(
      screen.getByRole('spinbutton', { name: /Remaining Quota/ })
    ).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Save changes' })).toBeDisabled()
    await act(async () =>
      resolveUser({ data: { success: true, data: target } })
    )
    expect(
      screen.getByRole('spinbutton', { name: /Remaining Quota/ })
    ).toBeEnabled()
    expect(screen.getByRole('button', { name: 'Save changes' })).toBeEnabled()
  })

  it('creates a user using the username as the compatible display name', async () => {
    const request = vi
      .spyOn(api, 'post')
      .mockResolvedValue({ data: { success: true } })
    const user = userEvent.setup()
    renderEditor()
    expect(screen.queryByLabelText('Display Name')).not.toBeInTheDocument()
    await user.type(
      screen.getByRole('textbox', { name: 'Username' }),
      'new-user'
    )
    await user.type(
      screen.getByLabelText('Password', { exact: true }),
      'NewPass123!'
    )
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() =>
      expect(request).toHaveBeenCalledWith('/api/user/', {
        username: 'new-user',
        display_name: 'new-user',
        password: 'NewPass123!',
        role: 1,
      })
    )
  })

  it('renders only the username in the user name column', () => {
    render(<UsernameColumn />)
    expect(screen.getByText('alice')).toBeVisible()
    expect(screen.queryByText('Legacy display')).not.toBeInTheDocument()
    expect(screen.queryByText('Existing note')).not.toBeInTheDocument()
  })
})
