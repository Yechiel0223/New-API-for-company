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
import { getCoreRowModel, useReactTable } from '@tanstack/react-table'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import type { User } from '../../types'
import { DataTableRowActions } from '../data-table-row-actions'
import { UsersDeleteDialog } from '../users-delete-dialog'
import { UsersProvider } from '../users-provider'

const target: User = {
  id: 7,
  username: 'alice',
  display_name: 'Alice',
  role: 1,
  status: 1,
  quota: 0,
  used_quota: 0,
  request_count: 0,
  group: 'default',
}

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

function Actions(props: { user: User }) {
  const table = useReactTable({
    data: [props.user],
    columns: [],
    getCoreRowModel: getCoreRowModel(),
  })
  return (
    <UsersProvider>
      <DataTableRowActions row={table.getRowModel().rows[0]} />
      <UsersDeleteDialog />
    </UsersProvider>
  )
}

describe('simplified user actions', () => {
  it.each([
    { status: 1, label: 'Disable', action: 'disable' },
    { status: 2, label: 'Enable', action: 'enable' },
  ])(
    'shows only $label and delete and sends the correct status action',
    async ({ status, label, action }) => {
      const request = vi
        .spyOn(api, 'post')
        .mockResolvedValue({ data: { success: true } })
      const user = userEvent.setup()
      render(<Actions user={{ ...target, status }} />)
      await user.click(screen.getByRole('button', { name: 'Open menu' }))
      expect(
        screen.getAllByRole('menuitem').map((item) => item.textContent)
      ).toEqual([label, 'Delete'])
      await user.click(screen.getByRole('menuitem', { name: label }))
      await waitFor(() =>
        expect(request).toHaveBeenCalledWith('/api/user/manage', {
          id: 7,
          action,
        })
      )
    }
  )

  it('requires confirmation before deleting the selected user', async () => {
    const request = vi
      .spyOn(api, 'delete')
      .mockResolvedValue({ data: { success: true } })
    const user = userEvent.setup()
    render(<Actions user={target} />)
    await user.click(screen.getByRole('button', { name: 'Open menu' }))
    await user.click(screen.getByRole('menuitem', { name: 'Delete' }))
    expect(await screen.findByRole('alertdialog')).toHaveTextContent('alice')
    expect(request).not.toHaveBeenCalled()
    await user.click(screen.getByRole('button', { name: 'Delete' }))
    await waitFor(() =>
      expect(request).toHaveBeenCalledExactlyOnceWith('/api/user/7/')
    )
  })

  it('keeps disable and delete unavailable for root users', async () => {
    const user = userEvent.setup()
    render(<Actions user={{ ...target, role: 100 }} />)
    await user.click(screen.getByRole('button', { name: 'Open menu' }))
    expect(screen.getByRole('menuitem', { name: 'Disable' })).toHaveAttribute(
      'aria-disabled',
      'true'
    )
    expect(screen.getByRole('menuitem', { name: 'Delete' })).toHaveAttribute(
      'aria-disabled',
      'true'
    )
  })
})
