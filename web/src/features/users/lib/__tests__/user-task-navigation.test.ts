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
import { describe, expect, test } from 'vitest'

import { ROLE } from '@/lib/roles'

import {
  buildUserTaskLogNavigation,
  canOpenUserTaskLogs,
} from '../user-task-navigation'

describe('user task log navigation', () => {
  test('allows admins and super admins to open a user task log list', () => {
    expect(canOpenUserTaskLogs(ROLE.ADMIN)).toBe(true)
    expect(canOpenUserTaskLogs(ROLE.SUPER_ADMIN)).toBe(true)
  })

  test('keeps non-admin users from opening user task logs from user management', () => {
    expect(canOpenUserTaskLogs(ROLE.USER)).toBe(false)
    expect(canOpenUserTaskLogs(ROLE.GUEST)).toBe(false)
  })

  test('always points user row clicks to task logs filtered by user id', () => {
    expect(buildUserTaskLogNavigation(42)).toEqual({
      to: '/usage-logs/$section',
      params: { section: 'task' },
      search: {
        page: 1,
        userId: 42,
      },
    })
  })
})
