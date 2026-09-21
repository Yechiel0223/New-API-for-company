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
import assert from 'node:assert/strict'

import { t } from 'i18next'
import { describe, test } from 'vitest'

import { ROLE } from '@/lib/roles'

import { resolveLogsViewAccess } from '../components/usage-logs-provider'
import {
  getVisibleUsageLogSectionsForRole,
  getUsageLogsSectionNavItems,
  resolveUsageLogsSectionForRole,
} from '../section-registry'

describe('usage log access tier', () => {
  test('keeps users and elevated self views on the self tier', () => {
    assert.equal(resolveLogsViewAccess(ROLE.USER, 'all'), 'self')
    assert.equal(resolveLogsViewAccess(ROLE.ADMIN, 'self'), 'self')
    assert.equal(resolveLogsViewAccess(ROLE.SUPER_ADMIN, 'self'), 'self')
  })

  test('distinguishes admin and root while viewing all logs', () => {
    assert.equal(resolveLogsViewAccess(ROLE.ADMIN, 'all'), 'admin')
    assert.equal(resolveLogsViewAccess(ROLE.SUPER_ADMIN, 'all'), 'root')
  })

  test('hides drawing logs for ordinary users', () => {
    assert.deepEqual(getVisibleUsageLogSectionsForRole(ROLE.USER), ['task'])
    assert.equal(resolveUsageLogsSectionForRole('drawing', ROLE.USER), 'task')
    assert.deepEqual(
      getUsageLogsSectionNavItems(t).map((item) => item.url),
      ['/usage-logs/task']
    )
  })

  test('limits ordinary admins to task logs only', () => {
    assert.deepEqual(getVisibleUsageLogSectionsForRole(ROLE.ADMIN), ['task'])
    assert.equal(resolveUsageLogsSectionForRole('drawing', ROLE.ADMIN), 'task')
    assert.equal(resolveUsageLogsSectionForRole('common', ROLE.ADMIN), 'task')
  })

  test('redirects removed common and drawing logs for root users', () => {
    assert.deepEqual(getVisibleUsageLogSectionsForRole(ROLE.SUPER_ADMIN), [
      'task',
    ])
    assert.equal(
      resolveUsageLogsSectionForRole('drawing', ROLE.SUPER_ADMIN),
      'task'
    )
    assert.equal(
      resolveUsageLogsSectionForRole('common', ROLE.SUPER_ADMIN),
      'task'
    )
  })
})
